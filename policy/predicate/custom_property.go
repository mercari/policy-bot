// Copyright 2025 Palantir Technologies, Inc.
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

package predicate

import (
	"context"
	"fmt"

	"github.com/palantir/policy-bot/policy/common"
	"github.com/palantir/policy-bot/pull"
	"github.com/pkg/errors"
)

type CustomProperty struct {
	Key string `yaml:"key,omitempty"`

	// This predicate evaluates to true if any of the following conditions are met.

	IsNull  bool `yaml:"is_null,omitempty"`
	NotNull bool `yaml:"not_null,omitempty"`

	// Only works for string values. For non-string values, any value specified here will result in the predicate failing.
	MatchesAnyOf []common.Regexp `yaml:"matches_any_of,omitempty"`

	// Evaluated before Matches.
	// Only works for string values. For non-string values, any value specified here will result in the predicate failing.
	MatchesNoneOf []common.Regexp `yaml:"matches_none_of,omitempty"`

	// Note: array support is not implemented yet.
	// Suggested implementation: ContainsAnyOf, ContainsAllOf, ContainsNoneOf
}

var _ Predicate = CustomProperty{}

func (pred CustomProperty) Evaluate(ctx context.Context, prctx pull.Context) (*common.PredicateResult, error) {
	predicateResult := common.PredicateResult{
		ValuePhrase:     fmt.Sprintf("custom property values with key \"%s\"", pred.Key),
		ConditionPhrase: "satisfy",
	}

	if pred.Key == "" {
		predicateResult.Satisfied = false
		predicateResult.Description = "Custom property key is unset"
		return &predicateResult, nil
	}

	// If no matches or not matches are provided, we assume that the custom properties are not required
	if !pred.IsNull && !pred.NotNull && len(pred.MatchesAnyOf) == 0 && len(pred.MatchesNoneOf) == 0 {
		predicateResult.Satisfied = false
		predicateResult.Description = "Custom property predicate is not configured to match any values"
		return &predicateResult, nil
	}

	customProperties, err := prctx.RepositoryCustomProperties()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository custom properties")
	}

	customProperty, ok := customProperties[pred.Key]
	if ok {
		if customProperty.String != nil {
			predicateResult.Values = []string{*customProperty.String}
		} else if customProperty.Array != nil {
			predicateResult.Values = customProperty.Array
		}
	}

	if pred.IsNull && !ok {
		predicateResult.Satisfied = true
		predicateResult.ConditionsMap = map[string][]string{"is null": {"true"}}
		predicateResult.Description = "Custom property is null"
		return &predicateResult, nil
	}
	if pred.NotNull && ok {
		predicateResult.Satisfied = true
		predicateResult.ConditionsMap = map[string][]string{"is not null": {"true"}}
		predicateResult.Description = "Custom property is not null"
		return &predicateResult, nil
	}

	var matchesAnyOfPatterns, matchesNoneOfPatterns []string

	for _, reg := range pred.MatchesAnyOf {
		matchesAnyOfPatterns = append(matchesAnyOfPatterns, reg.String())
	}

	for _, reg := range pred.MatchesNoneOf {
		matchesNoneOfPatterns = append(matchesNoneOfPatterns, reg.String())
	}

	if customProperty.String != nil {
		for _, matcher := range pred.MatchesAnyOf {
			if matcher.Matches(*customProperty.String) {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"matches any of": matchesAnyOfPatterns}
				predicateResult.Description = "Custom property matches one or more Matches patterns"
				return &predicateResult, nil
			}
		}

		if len(pred.MatchesNoneOf) > 0 {
			matched := false
			for _, matcher := range pred.MatchesNoneOf {
				if matcher.Matches(*customProperty.String) {
					matched = true
				}
			}
			if !matched {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"matches none of": matchesNoneOfPatterns}
				predicateResult.Description = "Custom property does not match any NotMatches pattern"
				return &predicateResult, nil
			}
		}
	}

	predicateResult.Satisfied = false
	predicateResult.Description = "Custom property does not satisfy any conditions configured"
	predicateResult.ConditionsMap = map[string][]string{}
	if pred.IsNull {
		predicateResult.ConditionsMap["is null"] = []string{"true"}
	}
	if pred.NotNull {
		predicateResult.ConditionsMap["is not null"] = []string{"true"}
	}
	if len(pred.MatchesAnyOf) > 0 {
		predicateResult.ConditionsMap["matches any of"] = matchesAnyOfPatterns
	}
	if len(pred.MatchesNoneOf) > 0 {
		predicateResult.ConditionsMap["matches none of"] = matchesNoneOfPatterns
	}
	return &predicateResult, nil
}

func (pred CustomProperty) Trigger() common.Trigger {
	// Ideally we would be able to trigger on custom properties change, but it is not possible right now
	return common.TriggerStatic
}
