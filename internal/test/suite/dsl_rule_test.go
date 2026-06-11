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
	// Registra a regra DSL de teste durante a inicialização do pacote
	rules.NewRule("test-dsl-rule").
		Name("Test DSL Rule").
		Description("Detecta chamadas a test-func com um número como argumento.").
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
	// Cria uma regra DSL que utiliza as configurações passadas no contexto
	rule := rules.NewRule("test-dsl-config-rule").
		When(func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
			val := rules.GetConfigInt(context, "test-dsl-config-rule", "test_key", 10)
			return val == 42
		}).
		Message("Config matched!").
		Register()

	node := &reader.RichNode{}

	// Caso 1: Sem arquivo/objeto de configuração no contexto
	// Deve retornar o padrão (10 != 42), fazendo o predicado falhar (retorna nil)
	finding1 := rule.Check(node, map[string]interface{}{}, "test.clj")
	assert.Nil(t, finding1)

	// Caso 2: Configuração presente, mas com valor diferente
	cfgWrong := &config.Config{
		RuleConfig: map[string]config.RuleSettings{
			"test-dsl-config-rule": {
				"test_key": 99,
			},
		},
	}
	findingWrong := rule.Check(node, map[string]interface{}{"config": cfgWrong}, "test.clj")
	assert.Nil(t, findingWrong)

	// Caso 3: Configuração correta presente (valor = 42)
	// Predicado deve passar e retornar o Finding correto
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
