# Framework de Testes para Regras de Análise

Este documento explica como usar o framework de testes personalizado do projeto Arit para criar e executar testes das regras de análise de código Clojure.

## 📁 Estrutura do Framework

```
internal/test/
├── framework/           # Framework base
│   └── framework.go    # Estruturas e funções principais
├── suite/              # Suítes de teste por regra
│   └── *_test.go      # Arquivos de teste Go
└── data/               # Arquivos de dados de teste
    └── *.clj          # Arquivos Clojure com casos de teste
```

## 🏗️ Componentes Principais

### 1. Framework Base (`framework/framework.go`)

#### `ExpectedFinding`
Define o que esperamos que uma regra encontre:
```go
type ExpectedFinding struct {
    Message   string  // Parte da mensagem esperada
    StartLine int     // Linha onde o problema deve ser detectado
}
```

#### `RuleTestCase`
Define um caso de teste completo:
```go
type RuleTestCase struct {
    FileToAnalyze    string             // Nome do arquivo .clj em data/
    RuleID           string             // ID da regra a ser testada
    ExpectedFindings []ExpectedFinding  // Lista de problemas esperados
}
```

#### `RunRuleTest()`
Função principal que executa o teste:
- Isola apenas a regra sendo testada
- Analisa o arquivo especificado
- Compara os resultados com as expectativas

## 🚀 Como Implementar um Novo Teste

### Passo 1: Criar o Arquivo de Dados (.clj)

Crie um arquivo em `internal/test/data/` com exemplos de código que sua regra deve detectar:

```clojure
;; Exemplo: data/minha_regra.clj
(ns minha-regra-test)

;; Caso 1: Problema que deve ser detectado na linha 4
(defn funcao-problematica []
  (+ 1 2 3))  ; <- Linha 4: problema aqui

;; Caso 2: Outro problema na linha 8  
(defn outra-funcao []
  (* 4 5 6))  ; <- Linha 8: outro problema

;; Caso 3: Código correto que NÃO deve ser detectado
(defn funcao-correta []
  (map inc [1 2 3]))
```

### Passo 2: Criar o Arquivo de Teste (.go)

Crie um arquivo em `internal/test/suite/` seguindo o padrão:

```go
// Exemplo: suite/minha_regra_test.go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestMinhaRegra(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "minha_regra.clj",           // Nome do arquivo em data/
            RuleID:        "minha-regra",               // ID da regra
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "problema detectado",     // Parte da mensagem esperada
                    StartLine: 4,                       // Linha do primeiro problema
                },
                {
                    Message:   "outro problema",        // Parte da mensagem do segundo
                    StartLine: 8,                       // Linha do segundo problema
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
```

### Passo 3: Executar os Testes

```bash
# Executar todos os testes (saída básica)
go test ./internal/test/suite/...

# Executar com saída detalhada e amigável (RECOMENDADO)
go test ./internal/test/suite/... -v

# Executar apenas um teste específico
go test ./internal/test/suite/ -run TestMinhaRegra -v

# Executar com cobertura de código
go test ./internal/test/suite/... -cover -v

# Gerar relatório HTML de cobertura
go test ./internal/test/suite/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 📝 Exemplo Completo: Regra `explicit-recursion`

### Arquivo de Dados (`data/explicit_recursion.clj`)
```clojure
(ns explicit-recursion)

;; Exemplo 1: Padrão 'map' que deve ser detectado
(defn double-nums-recursive [nums]
  (if (empty? nums)
    '()
    (cons (* 2 (first nums)) (double-nums-recursive (rest nums)))))

;; Exemplo 2: Padrão 'reduce' que deve ser detectado  
(defn sum-list [numbers]
  (if (empty? numbers)
    0
    (+ (first numbers) (sum-list (rest numbers)))))

;; Exemplo 3: Padrão 'filter' que deve ser detectado
(defn get-even-numbers [coll]
  (if (empty? coll)
    []
    (let [f (first coll)
          r (rest coll)]
      (if (even? f)
        (cons f (get-even-numbers r))
        (get-even-numbers r)))))
```

### Arquivo de Teste (`suite/explicit_recursion_test.go`)
```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestExplicitRecursion(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "explicit_recursion.clj",
            RuleID:        "explicit-recursion",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "transformation (map) pattern", StartLine: 4},
                {Message: "accumulator (reduce) pattern", StartLine: 10},
                {Message: "filtering pattern", StartLine: 16},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

## 🔍 Como Descobrir a Mensagem Esperada

### **Estratégia 1: Execução Manual (Recomendada)**
```bash
# Execute a ferramenta no seu arquivo de teste
./arit internal/test/data/sua_regra.clj

# A saída mostrará todas as mensagens, copie a parte mais significativa
```

### **Estratégia 2: Debug Test**
Use a função de debug para ver todas as mensagens encontradas:

```go
func TestMinhaRegra(t *testing.T) {
    testCase := framework.RuleTestCase{
        FileToAnalyze: "minha_regra.clj",
        RuleID:        "minha-regra",
        ExpectedFindings: []framework.ExpectedFinding{}, // Vazio primeiro
    }
    
    // Use DebugRuleTest em vez de RunRuleTest
    framework.DebugRuleTest(t, testCase)
}
```

### **Estratégia 3: Teste Vazio**
Crie o teste sem `ExpectedFindings` e veja a mensagem de erro:

```go
ExpectedFindings: []framework.ExpectedFinding{}, // Lista vazia
```

Execute o teste e ele mostrará quantos findings foram encontrados. Depois use a Estratégia 1 ou 2.

## 🎯 Dicas e Boas Práticas

### ✅ **DO's (Faça)**

1. **Nomes descritivos**: Use nomes claros para arquivos e funções
2. **Comentários explicativos**: Documente cada caso de teste no arquivo .clj
3. **Casos variados**: Inclua casos positivos (que devem ser detectados) e negativos (que não devem)
4. **Mensagens específicas**: Use partes específicas da mensagem no `ExpectedFinding.Message`
5. **Linhas corretas**: Verifique se `StartLine` corresponde à linha real do problema
6. **Teste primeiro**: Use as estratégias de descoberta antes de escrever os testes

### ❌ **DON'Ts (Não faça)**

1. **Não use mensagens genéricas**: Evite mensagens muito vagas como "erro"
2. **Não ignore casos negativos**: Sempre teste que código correto não é detectado
3. **Não hardcode caminhos**: Use apenas nomes de arquivos relativos
4. **Não teste múltiplas regras**: Cada teste deve focar em uma regra específica
5. **Não esqueça de executar**: Sempre execute os testes antes de fazer commit

## 🔧 Funcionalidades Avançadas

### Múltiplos Casos de Teste
```go
func TestMinhaRegra(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "caso_simples.clj",
            RuleID:        "minha-regra",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "problema simples", StartLine: 3},
            },
        },
        {
            FileToAnalyze: "caso_complexo.clj", 
            RuleID:        "minha-regra",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "problema complexo", StartLine: 5},
                {Message: "outro problema", StartLine: 10},
            },
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

### Testando Casos Sem Problemas
```go
{
    FileToAnalyze: "codigo_correto.clj",
    RuleID:        "minha-regra", 
    ExpectedFindings: []framework.ExpectedFinding{}, // Lista vazia = nenhum problema esperado
}
```

## 🎨 Saídas Mais Amigáveis

### **Opções do `go test` para Saídas Melhores**

```bash
# ✅ SEMPRE use -v para saída detalhada
go test ./internal/test/suite/... -v

# 🎯 Executa teste específico com detalhes
go test ./internal/test/suite/ -run TestMinhaRegra -v

# 📊 Mostra cobertura de código
go test ./internal/test/suite/... -cover -v

# 🚀 Execução paralela (mais rápido)
go test ./internal/test/suite/... -v -parallel=4

# ⏱️ Define timeout para testes longos
go test ./internal/test/suite/... -v -timeout=30s

# 📄 Saída em formato JSON (para ferramentas)
go test ./internal/test/suite/... -json
```

### **Saídas do Nosso Framework**

**✅ Quando testes passam:**
```
✅ Finding 1: OK - linha 4 contém "transformation (map) pattern"
✅ Finding 2: OK - linha 10 contém "accumulator (reduce) pattern"
```

**❌ Quando testes falham:**
```
❌ Finding 1 (linha 4): Mensagem não corresponde.
   Esperado (parte): "map pattern"
   Atual (completa): "transformation (filter) pattern. Consider using filter"
```

**🔍 Saída do Debug:**
```
🔍 === DEBUG: Findings encontrados para regra 'minha-regra' ===
📊 Total de findings: 2

🎯 Finding 1:
   📍 Linha: 4
   💬 Mensagem: "Parameter 'b' is unused"
   📝 Sugestão para teste:
      {Message: "Parameter 'b'", StartLine: 4},
```

## 🐛 Debugging de Testes

### Teste Falhou?

1. **Verifique a linha**: O problema está na linha esperada?
2. **Verifique a mensagem**: A mensagem contém o texto esperado?
3. **Execute com -v**: Use `go test -v` para ver detalhes
4. **Teste manualmente**: Execute `./arit arquivo.clj` para ver a saída real

### Mensagens de Erro Comuns

- **"Nenhum finding encontrado na linha X"**: A regra não detectou o problema na linha esperada
- **"A mensagem do finding não corresponde"**: A mensagem real não contém o texto esperado
- **"O número de findings não corresponde"**: A regra encontrou mais/menos problemas que o esperado

## 🛠️ Ferramentas Externas (Opcionais)

Se quiser ainda mais funcionalidades, considere estas ferramentas:

### **Testify** (Biblioteca de Assertions)
```bash
go get github.com/stretchr/testify
```
- ✅ Assertions mais expressivas
- ✅ Mocking avançado
- ✅ Suites de teste organizadas

### **GoConvey** (Interface Web)
```bash
go get github.com/smartystreets/goconvey
```
- ✅ Interface web para executar testes
- ✅ Atualizações em tempo real
- ✅ Relatórios visuais

### **Ginkgo + Gomega** (BDD Style)
```bash
go get github.com/onsi/ginkgo/v2
go get github.com/onsi/gomega
```
- ✅ Estilo Behavior-Driven Development
- ✅ Matchers poderosos
- ✅ Testes paralelos avançados

**Nota**: Nosso framework funciona perfeitamente com o `testing` padrão do Go, mas você pode integrar essas ferramentas se precisar de funcionalidades específicas.

## 📚 Referências

- [Documentação do pacote testing do Go](https://pkg.go.dev/testing)
- [Testify - biblioteca de assertions](https://github.com/stretchr/testify)
- [Boas práticas de teste em Go](https://go.dev/doc/code.html#Testing)
- [6 Golang Testing Frameworks](https://speedscale.com/blog/golang-testing-frameworks-for-every-type-of-test/)

## 🆘 Precisa de Ajuda?

Se encontrar dificuldades:

1. Verifique os exemplos existentes em `internal/test/suite/`
2. Execute os testes existentes para entender o padrão
3. Use `go test -v` para ver saídas detalhadas
4. Consulte a documentação das regras em `internal/rules/`

---

**Lembre-se**: Testes bem escritos são a base para um código confiável! 🚀 