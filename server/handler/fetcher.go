// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/go-github/v72/github"
	"github.com/palantir/go-githubapp/appconfig"
	"github.com/palantir/policy-bot/policy"
	"github.com/palantir/policy-bot/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"
)

type FetchedConfig struct {
	Config     *policy.Config
	LoadError  error
	ParseError error

	Source string
	Path   string
}

type ConfigCache struct {
	mu     sync.RWMutex
	expiry time.Time
	config *FetchedConfig
}

func (cc *ConfigCache) GetOrUpdate(fn func() (*FetchedConfig, error)) (*FetchedConfig, bool, error) {
	const cacheUpdateTimeout = 30 * time.Second
	const cacheExpiryDuration = 1 * time.Minute

	// Works both when the cache has not expired, and when another process is currently updating the cache (block on RLock).
	cached := func() *FetchedConfig {
		cc.mu.RLock()
		defer cc.mu.RUnlock()

		if time.Now().Before(cc.expiry) {
			return cc.config
		}
		return nil
	}()
	if cached != nil {
		return cached, true, nil
	}

	// Update
	cc.mu.Lock()
	defer cc.mu.Unlock()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), cacheUpdateTimeout)
	defer cancel()

	var value *FetchedConfig
	var err error
	var updateDone = make(chan struct{}, 1)
	go func() {
		defer close(updateDone)
		value, err = fn()
	}()

	select {
	case <-timeoutCtx.Done():
		return nil, false, errors.New("cache update timed out")
	case <-updateDone:
		if err != nil {
			return nil, false, err
		}
		cc.config = value
		cc.expiry = time.Now().Add(cacheExpiryDuration)
		return cc.config, false, nil
	}
}

type ConfigFetcher struct {
	sharedConfigCache ConfigCache

	Options PullEvaluationOptions
	Loader  *appconfig.Loader
}

func (cf *ConfigFetcher) configForSharedRepository(ctx context.Context, client *github.Client, owner string) (*FetchedConfig, error) {
	r, _, err := client.Repositories.Get(ctx, owner, *cf.Options.SharedRepository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", owner, *cf.Options.SharedRepository, err)
	}

	ref := r.GetDefaultBranch()
	file, _, _, err := client.Repositories.GetContents(ctx, owner, *cf.Options.SharedRepository, *cf.Options.SharedPolicyPath, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s/%s@%s:%s: %w", owner, *cf.Options.SharedRepository, ref, *cf.Options.SharedPolicyPath, err)
	}

	content, err := file.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get content of file %s/%s@%s:%s: %w", owner, *cf.Options.SharedRepository, ref, *cf.Options.SharedPolicyPath, err)
	}

	var pc policy.Config
	if err := yaml.UnmarshalStrict([]byte(content), &pc); err != nil {
		return nil, fmt.Errorf("failed to parse content of file %s/%s@%s:%s: %w", owner, *cf.Options.SharedRepository, ref, *cf.Options.SharedPolicyPath, err)
	}

	fc := &FetchedConfig{
		Config: &pc,
		Source: fmt.Sprintf("%s/%s@%s", owner, *cf.Options.SharedRepository, ref),
		Path:   *cf.Options.SharedPolicyPath,
	}
	return fc, nil
}

func (cf *ConfigFetcher) ConfigForSharedRepository(ctx context.Context, client *github.Client, owner string) FetchedConfig {
	ctx, span := tracing.Tracer.Start(ctx, "ConfigFetcher.ConfigForSharedRepository")
	defer span.End()

	config, cached, err := cf.sharedConfigCache.GetOrUpdate(func() (*FetchedConfig, error) {
		return cf.configForSharedRepository(ctx, client, owner)
	})
	span.SetAttributes(attribute.Bool("cache.hit", cached))

	if err != nil {
		return FetchedConfig{LoadError: err}
	}
	return *config
}

func (cf *ConfigFetcher) ConfigForRepositoryBranch(ctx context.Context, client *github.Client, owner, repository, branch string) FetchedConfig {
	ctx, span := tracing.Tracer.Start(ctx, "ConfigFetcher.ConfigForRepositoryBranch",
		trace.WithAttributes(
			attribute.String("owner", owner),
			attribute.String("repository", repository),
			attribute.String("branch", branch),
		))
	defer span.End()

	if cf.Options.ForceSharedPolicy {
		return cf.ConfigForSharedRepository(ctx, client, owner)
	}

	retries := 0
	delay := 1 * time.Second
	for {
		c, err := cf.Loader.LoadConfig(ctx, client, owner, repository, branch)
		fc := FetchedConfig{
			Source: c.Source,
			Path:   c.Path,
		}

		if err != nil {
			if !os.IsTimeout(err) && !isServerError(err) {
				fc.LoadError = err
				return fc
			}

			retries++
			if retries > 3 {
				fc.LoadError = err
				return fc
			}

			select {
			case <-ctx.Done():
				fc.LoadError = ctx.Err()
				return fc
			case <-time.After(delay):
				delay *= 2
				continue
			}
		}

		if c.IsUndefined() {
			return fc
		}

		var pc policy.Config
		if err := yaml.UnmarshalStrict(c.Content, &pc); err != nil {
			fc.ParseError = err
		} else {
			fc.Config = &pc
		}
		return fc
	}
}

func isServerError(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}
