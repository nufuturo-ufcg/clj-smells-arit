package suite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/test/framework"
)

func init() {
	// Register the test DSL rule during package initialization
	rules.NewRule("test-dsl-rule").
		Name("Test DSL Rule").
		Description("Detects calls to test-func with a number as an argument.").
		Severity(rules.SeverityInfo).
		When(rules.IsList()).
		When(rules.HasChildrenCount(2)).
		When(rules.FirstChildValueEquals("test-func")).
		When(rules.ChildMatches(1, rules.IsNumber())).
		Message("Detected test-func call with a number argument").
		Register()
}

func TestDSLRule(t *testing.T) {
	testCases := []framework.RuleTestCase{
		{
			FileToAnalyze: "dsl_test.clj",
			RuleID:        "test-dsl-rule",
			ExpectedFindings: []framework.ExpectedFinding{
				{
					Message:   "Detected test-func call with a number argument",
					StartLine: 3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.FileToAnalyze, func(t *testing.T) {
			framework.RunRuleTest(t, tc)
		})
	}
}

func TestDSLConfigLookup(t *testing.T) {
	// Create a DSL rule that uses the configuration supplied in the context
	rule := rules.NewRule("test-dsl-config-rule").
		When(func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
			val := rules.GetConfigInt(context, "test-dsl-config-rule", "test_key", 10)
			return val == 42
		}).
		Message("Config matched!").
		Register()

	node := &reader.RichNode{}

	// Case 1: No configuration file/object in the context
	// It should return the default (10 != 42), causing the predicate to fail (return nil)
	finding1 := rule.Check(node, map[string]interface{}{}, "test.clj")
	assert.Nil(t, finding1)

	// Case 2: Configuration is present but has a different value
	cfgWrong := &config.Config{
		RuleConfig: map[string]config.RuleSettings{
			"test-dsl-config-rule": {
				"test_key": 99,
			},
		},
	}
	findingWrong := rule.Check(node, map[string]interface{}{"config": cfgWrong}, "test.clj")
	assert.Nil(t, findingWrong)

	// Case 3: Correct configuration is present (value = 42)
	// The predicate should pass and return the correct Finding
	cfgCorrect := &config.Config{
		RuleConfig: map[string]config.RuleSettings{
			"test-dsl-config-rule": {
				"test_key": 42,
			},
		},
	}
	findingCorrect := rule.Check(node, map[string]interface{}{"config": cfgCorrect}, "test.clj")
	assert.NotNil(t, findingCorrect)
	assert.Equal(t, "Config matched!", findingCorrect.Message)
}

func TestNewPredicates(t *testing.T) {
	// 1. HasDescendant
	descendantRule := rules.NewRule("test-has-descendant").
		When(rules.HasDescendant(rules.ValueEquals("target-value"))).
		Message("Found target-value").
		Register()

	nodeWithDescendant := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{
				Type: reader.NodeList,
				Children: []*reader.RichNode{
					{Type: reader.NodeSymbol, Value: "target-value"},
				},
			},
		},
	}
	assert.NotNil(t, descendantRule.Check(nodeWithDescendant, map[string]interface{}{}, "test.clj"))

	// 2. ChildValueEquals and ChildIsSymbol
	childRule := rules.NewRule("test-child-shortcuts").
		When(rules.ChildValueEquals(0, "reset!")).
		When(rules.ChildIsSymbol(1)).
		Message("reset! shortcuts matched").
		Register()

	nodeShortcuts := &reader.RichNode{
		Type: reader.NodeList,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "reset!"},
			{Type: reader.NodeSymbol, Value: "my-atom"},
		},
	}
	assert.NotNil(t, childRule.Check(nodeShortcuts, map[string]interface{}{}, "test.clj"))

	// 3. HasBindingPair
	bindingRule := rules.NewRule("test-binding-pair").
		When(rules.HasBindingPair(rules.ValueEquals("x"), rules.IsNumber())).
		Message("binding matched").
		Register()

	bindingVector := &reader.RichNode{
		Type: reader.NodeVector,
		Children: []*reader.RichNode{
			{Type: reader.NodeSymbol, Value: "y"},
			{Type: reader.NodeString, Value: "hello"},
			{Type: reader.NodeSymbol, Value: "x"},
			{Type: reader.NodeNumber, Value: "42"},
		},
	}
	assert.NotNil(t, bindingRule.Check(bindingVector, map[string]interface{}{}, "test.clj"))
}
