package suite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
)

func TestNewAnalyzerGroupPriority(t *testing.T) {
	// Case 1: Only 'clojure-specific' rules enabled by default (empty config)
	cfg1 := &config.Config{
		EnabledRules:  make(map[string]bool),
		EnabledGroups: make(map[string]bool),
	}
	a1 := analyzer.NewAnalyzer(cfg1)
	assert.NotEmpty(t, a1.Rules)
	for _, rule := range a1.Rules {
		grp := rules.GetRuleGroup(rule.Meta().ID)
		assert.Equal(t, "clojure-specific", grp, "Only clojure-specific group rules should be enabled by default")
	}

	// Case 2: Disable the 'functional' group.
	// All rules under 'functional' (like 'explicit-recursion') must be disabled,
	// even if they are explicitly enabled in enabled-rules.
	cfg2 := &config.Config{
		EnabledGroups: map[string]bool{
			"functional": false,
		},
		EnabledRules: map[string]bool{
			"explicit-recursion": true,
		},
	}
	a2 := analyzer.NewAnalyzer(cfg2)

	for _, rule := range a2.Rules {
		assert.NotEqual(t, "explicit-recursion", rule.Meta().ID, "explicit-recursion should have been disabled by functional group disable")
	}

	// Case 3: Enable 'functional' group, but disable 'explicit-recursion' individually.
	// Since group enabling has highest priority, 'explicit-recursion' should be ENABLED.
	cfg3 := &config.Config{
		EnabledGroups: map[string]bool{
			"functional": true,
		},
		EnabledRules: map[string]bool{
			"explicit-recursion": false,
		},
	}
	a3 := analyzer.NewAnalyzer(cfg3)

	foundExplicitRecursion := false
	for _, rule := range a3.Rules {
		if rule.Meta().ID == "explicit-recursion" {
			foundExplicitRecursion = true
			break
		}
	}
	assert.True(t, foundExplicitRecursion, "explicit-recursion should have been enabled because its group was enabled")

	// Case 4: Group not specified, but rule disabled individually.
	// Should fall back to rule specification (disabled).
	cfg4 := &config.Config{
		EnabledGroups: make(map[string]bool),
		EnabledRules: map[string]bool{
			"explicit-recursion": false,
		},
	}
	a4 := analyzer.NewAnalyzer(cfg4)

	foundExplicitRecursion4 := false
	for _, rule := range a4.Rules {
		if rule.Meta().ID == "explicit-recursion" {
			foundExplicitRecursion4 = true
			break
		}
	}
	assert.False(t, foundExplicitRecursion4, "explicit-recursion should be disabled when not specified by group and explicitly disabled by rule")
}
