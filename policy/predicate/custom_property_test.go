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
	"regexp"
	"testing"

	"github.com/google/go-github/v72/github"
	"github.com/palantir/policy-bot/policy/common"
	"github.com/palantir/policy-bot/pull"
	"github.com/palantir/policy-bot/pull/pulltest"
	"github.com/stretchr/testify/assert"
)

func TestCustomPropertiesIsNullRule(t *testing.T) {
	p := &CustomProperty{
		Key:    "custom-property-key",
		IsNull: true,
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property is null",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    nil,
				ConditionsMap: map[string][]string{
					"is null": {"true"},
				},
			},
		},
		{
			"custom property is not null",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("value"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"value"},
				ConditionsMap: map[string][]string{
					"is null": {"true"},
				},
			},
		},
	})
}

func TestCustomPropertiesNotNullRule(t *testing.T) {
	p := &CustomProperty{
		Key:     "custom-property-key",
		NotNull: true,
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property is not null",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("value"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    []string{"value"},
				ConditionsMap: map[string][]string{
					"is not null": {"true"},
				},
			},
		},
		{
			"custom property is null",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    nil,
				ConditionsMap: map[string][]string{
					"is not null": {"true"},
				},
			},
		},
	})
}

func TestCustomPropertiesMatchesAnyOf(t *testing.T) {
	p := &CustomProperty{
		Key: "custom-property-key",
		MatchesAnyOf: []common.Regexp{
			common.NewCompiledRegexp(regexp.MustCompile("^value.*$")),
		},
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property matches pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("value123"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    []string{"value123"},
				ConditionsMap: map[string][]string{
					"matches any of": {"^value.*$"},
				},
			},
		},
		{
			"custom property does not match pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("other-value"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"other-value"},
				ConditionsMap: map[string][]string{
					"matches any of": {"^value.*$"},
				},
			},
		},
		{
			"custom property is not set",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    nil,
				ConditionsMap: map[string][]string{
					"matches any of": {"^value.*$"},
				},
			},
		},
	})
}

func TestCustomPropertiesMatchesNoneOf(t *testing.T) {
	p := &CustomProperty{
		Key: "custom-property-key",
		MatchesNoneOf: []common.Regexp{
			common.NewCompiledRegexp(regexp.MustCompile("^value.*$")),
		},
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property does not match pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("other-value"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    []string{"other-value"},
				ConditionsMap: map[string][]string{
					"matches none of": {"^value.*$"},
				},
			},
		},
		{
			"custom property matches pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("value123"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"value123"},
				ConditionsMap: map[string][]string{
					"matches none of": {"^value.*$"},
				},
			},
		},
		{
			"custom property is not set",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    nil,
				ConditionsMap: map[string][]string{
					"matches none of": {"^value.*$"},
				},
			},
		},
	})
}

type CustomPropertiesTestCase struct {
	name                    string
	context                 pull.Context
	ExpectedPredicateResult *common.PredicateResult
}

func runCustomPropertiesTestCase(t *testing.T, p Predicate, cases []CustomPropertiesTestCase) {
	ctx := context.Background()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			predicateResult, err := p.Evaluate(ctx, tc.context)
			if assert.NoError(t, err, "evaluation failed") {
				assertPredicateResult(t, tc.ExpectedPredicateResult, predicateResult)
			}
		})
	}
}
