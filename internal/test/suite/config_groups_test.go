package suite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
)

func TestNewAnalyzerGroupPriority(t *testing.T) {
	// Case 1: All rules enabled by default (empty config)
	cfg1 := &config.Config{
		EnabledRules:  make(map[string]bool),
		EnabledGroups: make(map[string]bool),
	}
	a1 := analyzer.NewAnalyzer(cfg1)
	assert.NotEmpty(t, a1.Rules)

	// Case 2: Disable the 'functional' group.
	// All rules under 'functional' (like 'immutability-violation') must be disabled,
	// even if they are explicitly enabled in enabled-rules.
	cfg2 := &config.Config{
		EnabledGroups: map[string]bool{
			"functional": false,
		},
		EnabledRules: map[string]bool{
			"immutability-violation": true,
		},
	}
	a2 := analyzer.NewAnalyzer(cfg2)

	for _, rule := range a2.Rules {
		assert.NotEqual(t, "immutability-violation", rule.Meta().ID, "immutability-violation should have been disabled by functional group disable")
	}

	// Case 3: Enable 'functional' group, but disable 'immutability-violation' individually.
	// Since group enabling has highest priority, 'immutability-violation' should be ENABLED.
	cfg3 := &config.Config{
		EnabledGroups: map[string]bool{
			"functional": true,
		},
		EnabledRules: map[string]bool{
			"immutability-violation": false,
		},
	}
	a3 := analyzer.NewAnalyzer(cfg3)

	foundImmutability := false
	for _, rule := range a3.Rules {
		if rule.Meta().ID == "immutability-violation" {
			foundImmutability = true
			break
		}
	}
	assert.True(t, foundImmutability, "immutability-violation should have been enabled because its group was enabled")

	// Case 4: Group not specified, but rule disabled individually.
	// Should fall back to rule specification (disabled).
	cfg4 := &config.Config{
		EnabledGroups: make(map[string]bool),
		EnabledRules: map[string]bool{
			"immutability-violation": false,
		},
	}
	a4 := analyzer.NewAnalyzer(cfg4)

	foundImmutability4 := false
	for _, rule := range a4.Rules {
		if rule.Meta().ID == "immutability-violation" {
			foundImmutability4 = true
			break
		}
	}
	assert.False(t, foundImmutability4, "immutability-violation should be disabled when not specified by group and explicitly disabled by rule")
}
