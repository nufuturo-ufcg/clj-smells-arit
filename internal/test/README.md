# Framework de Testes para Regras de Análise

Este documento contém a documentação completa do framework de testes personalizado do projeto Arit para criar e executar testes das regras de análise de código Clojure.

## Índice

1. [Visão Geral](#visão-geral)
2. [Estrutura do Framework](#estrutura-do-framework)
3. [Componentes Principais](#componentes-principais)
4. [Guia de Início Rápido](#guia-de-início-rápido)
5. [Como Implementar um Novo Teste](#como-implementar-um-novo-teste)
6. [Templates Prontos](#templates-prontos)
7. [Como Descobrir Mensagens Esperadas](#como-descobrir-mensagens-esperadas)
8. [Exemplo Prático Completo](#exemplo-prático-completo)
9. [Comandos e Execução](#comandos-e-execução)
10. [Depuração de Problemas](#depuração-de-problemas)
11. [Dicas e Boas Práticas](#dicas-e-boas-práticas)
12. [Referência Técnica](#referência-técnica)

---

## Visão Geral

O framework de testes foi desenvolvido para permitir testes automatizados das regras de análise de código. Ele isola regras específicas, executa análises em arquivos de teste e valida se os problemas esperados são detectados nas linhas corretas.

**Principais características:**
- Isolamento de regras individuais
- Validação de mensagens e localização
- Ferramentas de debug integradas
- Saída estruturada com testify
- Templates prontos para uso

---

## Estrutura do Framework

```
internal/test/
├── framework/           # Framework base
│   ├── framework.go    # Estruturas e funções principais
│   └── framework_debug.go  # Funções de debug
├── suite/              # Suítes de teste por regra
│   └── *_test.go      # Arquivos de teste Go
└── data/               # Arquivos de dados de teste
    └── *.clj          # Arquivos Clojure com casos de teste
```

---

## Componentes Principais

### 1. Estruturas de Dados

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

### 2. Funções Principais

#### `RunRuleTest(t *testing.T, tc RuleTestCase)`
Função principal que executa o teste:
- Isola apenas a regra sendo testada
- Analisa o arquivo especificado
- Compara os resultados com as expectativas
- Gera subtestes individuais para cada finding

#### `DebugRuleTest(t *testing.T, tc RuleTestCase)`
Função de debug que mostra todas as mensagens encontradas:
- Executa a regra no arquivo especificado
- Imprime todas as mensagens encontradas
- Sugere código pronto para copiar nos testes
- Útil para descobrir mensagens quando não as conhece

---

## Guia de Início Rápido

**Para desenvolvedores que querem implementar testes rapidamente!**

### Passo 1: Criar o arquivo de dados Clojure
**Arquivo**: `internal/test/data/minha_regra.clj`
```clojure
(ns minha-regra-test)

;; Código que DEVE ser detectado
(defn problema-aqui []
  (+ 1 2 3))  ; <- Linha 4: problema

;; Código que NÃO deve ser detectado  
(defn codigo-ok []
  (map inc [1 2 3]))
```

### Passo 2: Criar o arquivo de teste Go
**Arquivo**: `internal/test/suite/minha_regra_test.go`
```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestMinhaRegra(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "minha_regra.clj",
            RuleID:        "minha-regra",  // ID da sua regra
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "texto da mensagem",  // Parte da mensagem esperada
                    StartLine: 4,                    // Linha do problema
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

### Passo 3: Executar o teste
```bash
# SEMPRE use -v para saída detalhada!
go test ./internal/test/suite/ -run TestMinhaRegra -v
```

**Saída esperada:**
```
=== RUN   TestMinhaRegra
=== RUN   TestMinhaRegra/minha_regra.clj
=== RUN   TestMinhaRegra/minha_regra.clj/Finding_1_line_4
--- PASS: TestMinhaRegra (0.00s)
    --- PASS: TestMinhaRegra/minha_regra.clj (0.00s)
        --- PASS: TestMinhaRegra/minha_regra.clj/Finding_1_line_4 (0.00s)
```

### Checklist Rápido
- [ ] Arquivo `.clj` criado em `internal/test/data/`
- [ ] Arquivo `_test.go` criado em `internal/test/suite/`
- [ ] ID da regra correto
- [ ] Linhas contadas corretamente (começam em 1)
- [ ] Mensagem testada manualmente
- [ ] Teste executado com sucesso

---

## Como Implementar um Novo Teste

### Pré-requisitos
Antes de começar, certifique-se de que:
- [ ] A regra já está implementada em `internal/rules/`
- [ ] A regra está registrada no analisador principal
- [ ] Você conhece o ID da regra (ex: "minha-regra")
- [ ] Você tem exemplos de código que a regra deve detectar

### Passo 1: Criar o Arquivo de Dados (.clj)

Crie um arquivo em `internal/test/data/` with exemplos de código que sua regra deve detectar:

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
# Executar todos os testes
go test ./internal/test/suite/...

# Executar com saída detalhada (RECOMENDADO)
go test ./internal/test/suite/... -v

# Executar apenas um teste específico
go test ./internal/test/suite/ -run TestMinhaRegra -v

# Executar com cobertura de código
go test ./internal/test/suite/... -cover -v

# Gerar relatório HTML de cobertura
go test ./internal/test/suite/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Templates Prontos

### Template: Arquivo de Dados (.clj)

**Nome**: `internal/test/data/NOME_DA_REGRA.clj`

```clojure
(ns NOME-DA-REGRA-test)

;; ===========================================
;; CASOS POSITIVOS (devem ser detectados)
;; ===========================================

;; Caso 1: Descrição do problema
(defn exemplo-problema-1 []
  ;; Código que deve ser detectado
  (+ 1 2 3))  ; <- Linha X: descreva o problema aqui

;; Caso 2: Outro tipo de problema  
(defn exemplo-problema-2 []
  ;; Outro código problemático
  (* 4 5 6))  ; <- Linha Y: outro problema

;; ===========================================
;; CASOS NEGATIVOS (NÃO devem ser detectados)
;; ===========================================

;; Exemplo de código correto
(defn exemplo-correto []
  ;; Este código está OK e não deve gerar alertas
  (map inc [1 2 3]))

;; Outro exemplo correto
(defn outro-exemplo-correto []
  (reduce + [1 2 3 4]))
```

### Template: Arquivo de Teste (.go)

**Nome**: `internal/test/suite/NOME_DA_REGRA_test.go`

```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestNOME_DA_REGRA(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "NOME_DA_REGRA.clj",
            RuleID:        "NOME-DA-REGRA",  // ID exato da regra
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "PARTE_DA_MENSAGEM_ESPERADA",  // Ex: "problema detectado"
                    StartLine: X,  // Número da linha onde está o problema
                },
                {
                    Message:   "OUTRA_PARTE_DA_MENSAGEM",
                    StartLine: Y,  // Linha do segundo problema
                },
                // Adicione mais findings conforme necessário
            },
        },
        // Você pode adicionar mais casos de teste aqui se necessário
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

---

## Como Descobrir Mensagens Esperadas

### Método 1: Execução Manual (Recomendado)
```bash
# Execute a ferramenta no seu arquivo de teste
./arit internal/test/data/sua_regra.clj

# A saída mostrará todas as mensagens, copie a parte mais significativa
```

### Método 2: Debug Test
Use `framework.DebugRuleTest()` em vez de `framework.RunRuleTest()`:

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

**Execute o debug:**
```bash
go test ./internal/test/suite/ -run TestMinhaRegra -v
```

**Saída esperada:**
```
--- DEBUG: Findings for rule 'minha-regra' ---
Total findings: 2

Finding 1:
   Line: 4
   Message: "Parameter 'b' in function 'funcao-problema' is declared but never used"
   Suggested for test:
      {Message: "Parameter 'b'", StartLine: 4},

Finding 2:
   Line: 9
   Message: "Variable 'y' in let binding is declared but never used"
   Suggested for test:
      {Message: "Variable 'y'", StartLine: 9},
--- END DEBUG ---
```

**Perfeito!** O debug já sugere o código pronto para copiar!

### Método 3: Teste Vazio
Deixe `ExpectedFindings: []` vazio e execute o teste para ver quantos findings existem.

### Dicas para Mensagens

#### 1. Mensagens muito longas?
Se a mensagem for muito longa, use apenas a parte mais significativa:

```
Mensagem real: "Parameter 'b' in function 'funcao-problema' is declared but never used. Consider removing it or using it."

Use apenas: "Parameter 'b'"
```

#### 2. Múltiplas variações?
Se sua regra gera mensagens ligeiramente diferentes, escolha a parte comum:

```
"Parameter 'a' is unused"
"Parameter 'b' is unused"  
"Parameter 'x' is unused"

Use: "is unused"
```

#### 3. Case-sensitive!
Lembre-se que a comparação é case-sensitive:

```go
// Vai falhar
{Message: "PARAMETER 'b'", StartLine: 4}

// Vai passar  
{Message: "Parameter 'b'", StartLine: 4}
```

---

## Exemplo Prático Completo

### Cenário: Nova Regra "unused-variable"

Imagine que você implementou uma regra que detecta variáveis não utilizadas, mas não sabe exatamente quais mensagens ela gera.

### Passo 1: Criar o arquivo de dados

```clojure
;; internal/test/data/unused_variable.clj
(ns unused-variable-test)

;; Caso com variável não usada
(defn funcao-problema [a b]  ; b não é usado
  (* a 2))

;; Caso com let e variável não usada  
(defn outra-funcao []
  (let [x 10
        y 20]  ; y não é usado
    x))
```

### Passo 2: Execução Manual

```bash
$ ./arit internal/test/data/unused_variable.clj
[WARN] unused-variable: Parameter 'b' in function 'funcao-problema' is declared but never used [unused_variable.clj:4:1]
[WARN] unused-variable: Variable 'y' in let binding is declared but never used [unused_variable.clj:9:8]
```

**Resultado**: Agora você sabe as mensagens!

### Passo 3: Implementar o teste final

```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnusedVariable(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "unused_variable.clj",
            RuleID:        "unused-variable",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "Parameter 'b'", StartLine: 4},
                {Message: "Variable 'y'", StartLine: 9},
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

---

## Comandos e Execução

### Comandos Básicos
```bash
# Executar apenas seu teste
go test ./internal/test/suite/ -run TestNOME_DA_REGRA

# Executar com saída detalhada
go test ./internal/test/suite/ -run TestNOME_DA_REGRA -v

# Executar todos os testes
go test ./internal/test/suite/...

# Testar manualmente sua regra
./arit internal/test/data/NOME_DA_REGRA.clj
```

### Comandos Avançados
```bash
# Executar com cobertura
go test ./internal/test/suite/... -cover -v

# Gerar relatório HTML de cobertura
go test ./internal/test/suite/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Executar com timeout
go test ./internal/test/suite/... -v -timeout=30s

# Executar em paralelo
go test ./internal/test/suite/... -v -parallel=4
```

### Saída dos Testes

O framework usa a biblioteca testify para gerar saídas estruturadas. Quando executado com `-v`, você verá:

```
=== RUN   TestExplicitRecursion
=== RUN   TestExplicitRecursion/explicit_recursion.clj
=== RUN   TestExplicitRecursion/explicit_recursion.clj/Finding_1_line_6
=== RUN   TestExplicitRecursion/explicit_recursion.clj/Finding_2_line_12
--- PASS: TestExplicitRecursion (0.00s)
    --- PASS: TestExplicitRecursion/explicit_recursion.clj (0.00s)
        --- PASS: TestExplicitRecursion/explicit_recursion.clj/Finding_1_line_6 (0.00s)
        --- PASS: TestExplicitRecursion/explicit_recursion.clj/Finding_2_line_12 (0.00s)
```

Cada finding é testado individualmente como um subteste, permitindo identificar exatamente qual validação falhou.

---

## Depuração de Problemas

### Teste Falha: "Expected finding on line X, but none was found"
**Possíveis causas:**
- Linha incorreta no teste
- Regra não detecta o problema esperado
- Arquivo de dados incorreto

**Soluções:**
- Verifique se a linha está correta (editores mostram números de linha)
- Execute manualmente: `./arit internal/test/data/seu_arquivo.clj`
- Use `DebugRuleTest` para ver todos os findings

### Teste Falha: "Finding message does not contain expected text"
**Possíveis causas:**
- Mensagem esperada não corresponde à real
- Case-sensitive
- Mensagem muito específica

**Soluções:**
- Execute manualmente para ver a mensagem real
- Use apenas parte da mensagem, não a mensagem completa
- Verifique case-sensitive

### Teste Falha: "Incorrect number of findings"
**Possíveis causas:**
- Casos negativos sendo detectados erroneamente
- Casos positivos não sendo detectados
- Contagem incorreta

**Soluções:**
- Use `DebugRuleTest` para ver quantos findings existem
- Verifique se há casos negativos sendo detectados erroneamente
- Confirme se todos os casos positivos estão sendo detectados

### Fluxo de Debug Recomendado

1. **Crie o arquivo .clj** com casos de teste
2. **Execute manualmente** `./arit arquivo.clj`
3. **Se não funcionar**, use `DebugRuleTest()`
4. **Copie as sugestões** geradas pelo debug
5. **Substitua por `RunRuleTest()`** no teste final
6. **Execute o teste** para confirmar que passa

### Problemas Comuns

| Erro | Solução |
|------|---------|
| "Expected finding on line X, but none was found" | Verifique se a linha está correta |
| "Finding message does not contain expected text" | Execute manualmente e copie parte da mensagem real |
| "Incorrect number of findings" | Conte quantos problemas sua regra realmente detecta |

---

## Dicas e Boas Práticas

### **DO's (Faça)**

1. **Nomes descritivos**: Use nomes claros para arquivos e funções
2. **Comentários explicativos**: Documente cada caso de teste no arquivo .clj
3. **Casos variados**: Inclua casos positivos (que devem ser detectados) e negativos (que não devem)
4. **Mensagens específicas**: Use partes específicas da mensagem no `ExpectedFinding.Message`
5. **Linhas corretas**: Verifique se `StartLine` corresponde à linha real do problema
6. **Teste primeiro**: Use as estratégias de descoberta antes de escrever os testes

### **DON'Ts (Não faça)**

1. **Não use mensagens genéricas**: Evite mensagens muito vagas como "erro"
2. **Não ignore casos negativos**: Sempre teste que código correto não é detectado
3. **Não hardcode caminhos**: Use apenas nomes de arquivos relativos
4. **Não teste múltiplas regras**: Cada teste deve focar em uma regra específica
5. **Não esqueça de executar**: Sempre execute os testes antes de fazer commit

### Dicas de Implementação

#### 1. Como descobrir a mensagem exata da regra?
```bash
# Execute a ferramenta no seu arquivo de teste
./arit internal/test/data/sua_regra.clj

# A saída mostrará a mensagem real, use parte dela no teste
```

#### 2. Como contar as linhas corretamente?
- Use um editor que mostre números de linha
- Lembre-se: a primeira linha é 1, não 0
- Conte a linha onde o PROBLEMA está, não onde começa a função

#### 3. Como testar casos sem problemas?
```go
{
    FileToAnalyze: "codigo_limpo.clj",
    RuleID:        "sua-regra",
    ExpectedFindings: []framework.ExpectedFinding{}, // Lista vazia!
}
```

#### 4. Como debuggar testes que falham?

1. **Execute manualmente primeiro**:
   ```bash
   ./arit internal/test/data/seu_arquivo.clj
   ```

2. **Compare a saída real com o esperado**:
    - Mensagem: contém o texto esperado?
    - Linha: está na linha correta?

3. **Execute o teste com -v**:
   ```bash
   go test ./internal/test/suite/ -run SeuTeste -v
   ```

4. **Use DebugRuleTest**:
   ```go
   // Substitua temporariamente RunRuleTest por DebugRuleTest
   framework.DebugRuleTest(t, tc)
   ```

### Erros Comuns

1. **Nome do arquivo errado**: Certifique-se de que o nome em `FileToAnalyze` corresponde ao arquivo real
2. **ID da regra errado**: Use exatamente o mesmo ID definido na implementação da regra
3. **Linha errada**: Conte as linhas corretamente (editores ajudam!)
4. **Mensagem muito específica**: Use apenas parte da mensagem, não a mensagem completa
5. **Esquecer casos negativos**: Sempre teste que código correto não gera alertas

---

## Referência Técnica

### Funcionalidades Avançadas

#### Múltiplos Casos de Teste
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

#### Testando Casos Negativos (Sem Problemas)
```go
{
    FileToAnalyze: "codigo_limpo.clj",
    RuleID:        "minha-regra",
    ExpectedFindings: []framework.ExpectedFinding{}, // Lista vazia
}
```

### Estrutura de Validação

O framework valida:

1. **Número correto de findings**: Deve corresponder ao número de `ExpectedFindings`
2. **Linha correta**: Cada finding deve estar na linha especificada
3. **Mensagem correta**: Deve conter o texto especificado em `Message`
4. **RuleID correto**: Deve corresponder ao `RuleID` do teste

### Checklist de Debug

- [ ] Arquivo .clj criado com casos representativos
- [ ] Execução manual testada
- [ ] Debug test executado (se necessário)
- [ ] Mensagens copiadas corretamente
- [ ] Linhas verificadas (começam em 1)
- [ ] Teste final executado com sucesso
- [ ] `DebugRuleTest` removido/comentado

---

## Conclusão

Este framework fornece uma maneira estruturada e confiável de testar regras de análise. Use as estratégias de descoberta para encontrar mensagens, implemente casos variados e sempre execute os testes antes de fazer commits.

**Lembre-se**:
- Comece simples e vá incrementando
- É melhor ter poucos casos bem testados do que muitos casos mal implementados
- O debug é uma ferramenta temporária - use para descobrir mensagens e depois volte para `RunRuleTest()`
- Teste tanto casos positivos quanto negativos

Para dúvidas ou problemas, consulte os arquivos de exemplo em `internal/test/suite/` ou use as funções de debug disponíveis. 