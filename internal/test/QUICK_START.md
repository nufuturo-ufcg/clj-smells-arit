# 🚀 Guia de Início Rápido - Framework de Testes

**Para desenvolvedores que querem implementar testes rapidamente!**

## ⚡ Em 5 Minutos

### 1️⃣ Crie o arquivo de dados Clojure
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

### 2️⃣ Crie o arquivo de teste Go
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

### 3️⃣ Execute o teste
```bash
# SEMPRE use -v para saída amigável!
go test ./internal/test/suite/ -run TestMinhaRegra -v
```

**Saída esperada:**
```
=== RUN   TestMinhaRegra
✅ Finding 1: OK - linha 4 contém "texto da mensagem"
--- PASS: TestMinhaRegra (0.00s)
```

## 🔧 Como descobrir a mensagem?

### **Método 1: Execução Manual**
```bash
./arit internal/test/data/minha_regra.clj
```

### **Método 2: Debug Test**
Use `framework.DebugRuleTest()` em vez de `framework.RunRuleTest()`:

```go
// Substitua temporariamente esta linha:
// framework.RunRuleTest(t, tc)

// Por esta:
framework.DebugRuleTest(t, tc)
```

Isso imprimirá todas as mensagens encontradas com sugestões prontas para copiar!

### **Método 3: Teste Vazio**
Deixe `ExpectedFindings: []` vazio e execute o teste para ver quantos findings existem.

## ✅ Checklist Final

- [ ] Arquivo `.clj` criado em `internal/test/data/`
- [ ] Arquivo `_test.go` criado em `internal/test/suite/`
- [ ] ID da regra correto
- [ ] Linhas contadas corretamente (começam em 1)
- [ ] Mensagem testada manualmente
- [ ] Teste executado com sucesso

## 🆘 Problemas Comuns

| Erro | Solução |
|------|---------|
| "Nenhum finding encontrado na linha X" | Verifique se a linha está correta |
| "A mensagem não corresponde" | Execute manualmente e copie parte da mensagem real |
| "Número de findings não corresponde" | Conte quantos problemas sua regra realmente detecta |

## 📚 Precisa de mais detalhes?

- Leia `internal/test/README.md` para documentação completa
- Veja `internal/test/TEMPLATE.md` para templates detalhados
- Consulte `internal/test/suite/explicit_recursion_test.go` para exemplo real

---

**Agora é só implementar! 🎉** 