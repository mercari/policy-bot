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
	p := &CustomProperties{
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
				Values:    []string{"custom-property-key"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"is null": {"true"},
				},
			},
		},
	})
}

func TestCustomPropertiesNotNullRule(t *testing.T) {
	p := &CustomProperties{
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
				Values:    []string{"custom-property-key"},
			},
		},
		{
			"custom property is null",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not null": {"true"},
				},
			},
		},
	})
}

func TestCustomPropertiesMatchesRule(t *testing.T) {
	p := &CustomProperties{
		Key: "custom-property-key",
		Matches: []common.Regexp{
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"matches": {"^value.*$"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"matches": {"^value.*$"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"matches": {"^value.*$"},
				},
			},
		},
	})
}

func TestCustomPropertiesNotMatchesRule(t *testing.T) {
	p := &CustomProperties{
		Key: "custom-property-key",
		NotMatches: []common.Regexp{
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not matches": {"^value.*$"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not matches": {"^value.*$"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not matches": {"^value.*$"},
				},
			},
		},
	})
}

func TestCustomPropertiesContainsRule(t *testing.T) {
	p := &CustomProperties{
		Key:      "custom-property-key",
		Contains: []string{"value"},
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property contains pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						Array: []string{"value", "other-value"},
					},
				},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"contains": {"value"},
				},
			},
		},
		{
			"custom property does not contain pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						Array: []string{"other-value"},
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"contains": {"value"},
				},
			},
		},
		{
			"custom property is of wrong type",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						String: github.Ptr("value"),
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"contains": {"value"},
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
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"contains": {"value"},
				},
			},
		},
	})
}

func TestCustomPropertiesNotContainsRule(t *testing.T) {
	p := &CustomProperties{
		Key:         "custom-property-key",
		NotContains: []string{"value"},
	}

	runCustomPropertiesTestCase(t, p, []CustomPropertiesTestCase{
		{
			"custom property does not contain pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						Array: []string{"other-value"},
					},
				},
			},
			&common.PredicateResult{
				Satisfied: true,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not contains": {"value"},
				},
			},
		},
		{
			"custom property contains pattern",
			&pulltest.Context{
				RepositoryCustomPropertiesValue: map[string]pull.CustomProperty{
					"custom-property-key": {
						Array: []string{"value", "other-value"},
					},
				},
			},
			&common.PredicateResult{
				Satisfied: false,
				Values:    []string{"custom-property-key"},
				ConditionsMap: map[string][]string{
					"not contains": {"value"},
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
