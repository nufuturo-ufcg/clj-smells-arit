# Template para Implementar Novos Testes

Este arquivo contém templates prontos para usar ao implementar testes para novas regras.

## 📋 Checklist Rápido

Antes de começar, certifique-se de que:
- [ ] A regra já está implementada em `internal/rules/`
- [ ] A regra está registrada no analisador principal
- [ ] Você conhece o ID da regra (ex: "minha-regra")
- [ ] Você tem exemplos de código que a regra deve detectar

## 🚀 Template Rápido

### 1. Arquivo de Dados (.clj)

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

### 2. Arquivo de Teste (.go)

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

## 📝 Exemplo Prático: Regra "unused-variable"

Vamos implementar um teste para uma regra hipotética que detecta variáveis não utilizadas:

### Arquivo: `internal/test/data/unused_variable.clj`

```clojure
(ns unused-variable-test)

;; Caso 1: Variável não utilizada em let
(defn funcao-com-variavel-nao-usada []
  (let [x 10
        y 20]  ; <- y não é usado
    x))

;; Caso 2: Parâmetro não utilizado
(defn funcao-com-parametro-nao-usado [a b]  ; <- b não é usado
  (* a 2))

;; Caso 3: Código correto - todas as variáveis são usadas
(defn funcao-correta [a b]
  (+ a b))

;; Caso 4: Código correto - let com todas as variáveis usadas
(defn outra-funcao-correta []
  (let [x 10
        y 20]
    (+ x y)))
```

### Arquivo: `internal/test/suite/unused_variable_test.go`

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
                {
                    Message:   "Variable 'y' is declared but never used",
                    StartLine: 5,  // Linha onde 'y' é declarado
                },
                {
                    Message:   "Parameter 'b' is declared but never used", 
                    StartLine: 9,  // Linha onde 'b' é declarado
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

## 🔧 Comandos Úteis

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

## 🎯 Dicas de Implementação

### 1. Como descobrir a mensagem exata da regra?
```bash
# Execute a ferramenta no seu arquivo de teste
./arit internal/test/data/sua_regra.clj

# A saída mostrará a mensagem real, use parte dela no teste
```

### 2. Como contar as linhas corretamente?
- Use um editor que mostre números de linha
- Lembre-se: a primeira linha é 1, não 0
- Conte a linha onde o PROBLEMA está, não onde começa a função

### 3. Como testar casos sem problemas?
```go
{
    FileToAnalyze: "codigo_limpo.clj",
    RuleID:        "sua-regra",
    ExpectedFindings: []framework.ExpectedFinding{}, // Lista vazia!
}
```

### 4. Como debuggar testes que falham?

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

## ⚠️ Erros Comuns

1. **Nome do arquivo errado**: Certifique-se de que o nome em `FileToAnalyze` corresponde ao arquivo real
2. **ID da regra errado**: Use exatamente o mesmo ID definido na implementação da regra
3. **Linha errada**: Conte as linhas corretamente (editores ajudam!)
4. **Mensagem muito específica**: Use apenas parte da mensagem, não a mensagem completa
5. **Esquecer casos negativos**: Sempre teste que código correto não gera alertas

## 🚀 Próximos Passos

1. Copie os templates acima
2. Substitua `NOME_DA_REGRA` pelo nome real da sua regra
3. Implemente os casos de teste no arquivo .clj
4. Ajuste as mensagens e linhas esperadas no arquivo .go
5. Execute os testes e ajuste conforme necessário
6. Commit e push! 🎉

---

**Lembre-se**: Comece simples e vá incrementando. É melhor ter poucos casos bem testados do que muitos casos mal implementados! 