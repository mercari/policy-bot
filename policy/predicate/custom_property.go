package predicate

import (
	"context"
	"fmt"
	"slices"

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
	Matches []common.Regexp `yaml:"matches,omitempty"`

	// Evaluated before Matches.
	// Only works for string values. For non-string values, any value specified here will result in the predicate failing.
	NotMatches []common.Regexp `yaml:"not_matches,omitempty"`

	// Only works for array values. For non-array values, any value specified here will result in the predicate failing.
	Contains []string `yaml:"contains,omitempty"`

	// Evaluated before Contains.
	// Only works for array values. For non-array values, any value specified here will result in the predicate failing.
	NotContains []string `yaml:"not_contains,omitempty"`
}

var _ Predicate = CustomProperty{}

func (pred CustomProperty) Evaluate(ctx context.Context, prctx pull.Context) (*common.PredicateResult, error) {
	predicateResult := common.PredicateResult{
		ValuePhrase: fmt.Sprintf("custom property values with key \"%s\"", pred.Key),
	}

	if pred.Key == "" {
		predicateResult.Satisfied = false
		predicateResult.Description = "Custom property key is unset"
		return &predicateResult, nil
	}

	// If no matches or not matches are provided, we assume that the custom properties are not required
	if !pred.IsNull && !pred.NotNull && len(pred.Matches) == 0 && len(pred.NotMatches) == 0 && len(pred.Contains) == 0 && len(pred.NotContains) == 0 {
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
		predicateResult.ConditionPhrase = "is null"
		predicateResult.Description = "Custom property is null"
		return &predicateResult, nil
	}
	if pred.NotNull && ok {
		predicateResult.Satisfied = true
		predicateResult.ConditionPhrase = "is not null"
		predicateResult.Description = "Custom property is not null"
		return &predicateResult, nil
	}

	var matchesPatterns, notMatchesPatterns []string

	for _, reg := range pred.Matches {
		matchesPatterns = append(matchesPatterns, reg.String())
	}

	for _, reg := range pred.NotMatches {
		notMatchesPatterns = append(notMatchesPatterns, reg.String())
	}

	if customProperty.String != nil {
		for _, matcher := range pred.Matches {
			if matcher.Matches(*customProperty.String) {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"matches any of": matchesPatterns}
				predicateResult.Description = "Custom property matches one or more Matches patterns"
				return &predicateResult, nil
			}
		}

		if len(pred.NotMatches) > 0 {
			matched := false
			for _, matcher := range pred.NotMatches {
				if matcher.Matches(*customProperty.String) {
					matched = true
				}
			}
			if !matched {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"matches none of": notMatchesPatterns}
				predicateResult.Description = "Custom property does not match any NotMatches pattern"
				return &predicateResult, nil
			}
		}
	}

	if customProperty.Array != nil {
		for _, contains := range pred.Contains {
			if slices.Contains(customProperty.Array, contains) {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"contains any of": pred.Contains}
				predicateResult.Description = "Custom property contains one or more items in the Contains list"
				return &predicateResult, nil
			}
		}

		if len(pred.NotContains) > 0 {
			contains := false
			for _, notContains := range pred.NotContains {
				if slices.Contains(customProperty.Array, notContains) {
					contains = true
				}
			}
			if !contains {
				predicateResult.Satisfied = true
				predicateResult.ConditionsMap = map[string][]string{"contains none of": pred.NotContains}
				predicateResult.Description = "Custom property does not contain any items from the NotContains list"
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
	if len(pred.Matches) > 0 {
		predicateResult.ConditionsMap["matches any of"] = matchesPatterns
	}
	if len(pred.NotMatches) > 0 {
		predicateResult.ConditionsMap["matches none of"] = notMatchesPatterns
	}
	if len(pred.Contains) > 0 {
		predicateResult.ConditionsMap["contains any of"] = pred.Contains
	}
	if len(pred.NotContains) > 0 {
		predicateResult.ConditionsMap["contains none of"] = pred.NotContains
	}
	return &predicateResult, nil
}

// Ideally we would be able to trigger on custom properties change, but it is not possible right now
func (pred CustomProperty) Trigger() common.Trigger {
	return common.TriggerStatic
}
