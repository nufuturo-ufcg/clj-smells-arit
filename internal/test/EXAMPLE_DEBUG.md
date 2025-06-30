# 🔍 Exemplo Prático: Descobrindo Mensagens Esperadas

Este documento mostra **passo a passo** como descobrir as mensagens esperadas quando você não as conhece.

## 📝 Cenário: Nova Regra "unused-variable"

Imagine que você implementou uma regra que detecta variáveis não utilizadas, mas não sabe exatamente quais mensagens ela gera.

### **Passo 1: Criar o arquivo de dados**

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

### **Passo 2: Estratégia 1 - Execução Manual**

```bash
$ ./arit internal/test/data/unused_variable.clj
[WARN] unused-variable: Parameter 'b' in function 'funcao-problema' is declared but never used [unused_variable.clj:4:1]
[WARN] unused-variable: Variable 'y' in let binding is declared but never used [unused_variable.clj:9:8]
```

**Resultado**: Agora você sabe as mensagens! 🎉

### **Passo 3: Estratégia 2 - Debug Test** 

Se a execução manual não funcionar ou você quiser mais detalhes:

```go
// internal/test/suite/unused_variable_test.go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnusedVariable(t *testing.T) {
    testCase := framework.RuleTestCase{
        FileToAnalyze: "unused_variable.clj",
        RuleID:        "unused-variable",
        ExpectedFindings: []framework.ExpectedFinding{}, // Vazio primeiro!
    }
    
    // Use DebugRuleTest para descobrir as mensagens
    framework.DebugRuleTest(t, testCase)
}
```

**Execute o debug:**
```bash
go test ./internal/test/suite/ -run TestUnusedVariable -v
```

**Saída esperada:**
```
=== DEBUG: Findings encontrados para regra 'unused-variable' ===
Total de findings: 2

Finding 1:
  Linha: 4
  Mensagem: "Parameter 'b' in function 'funcao-problema' is declared but never used"
  Sugestão para teste:
    {Message: "Parameter 'b'", StartLine: 4},

Finding 2:
  Linha: 9
  Mensagem: "Variable 'y' in let binding is declared but never used"
  Sugestão para teste:
    {Message: "Variable 'y'", StartLine: 9},
=== FIM DEBUG ===
```

**Perfeito!** O debug já sugere o código pronto para copiar! 🚀

### **Passo 4: Implementar o teste final**

```go
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
            // Volte para RunRuleTest quando souber as mensagens
            framework.RunRuleTest(t, tc)
        })
    }
}
```

## 🎯 **Dicas Avançadas**

### **1. Mensagens muito longas?**
Se a mensagem for muito longa, use apenas a parte mais significativa:

```
Mensagem real: "Parameter 'b' in function 'funcao-problema' is declared but never used. Consider removing it or using it."

Use apenas: "Parameter 'b'"
```

### **2. Múltiplas variações?**
Se sua regra gera mensagens ligeiramente diferentes, escolha a parte comum:

```
"Parameter 'a' is unused"
"Parameter 'b' is unused"  
"Parameter 'x' is unused"

Use: "is unused"
```

### **3. Case-sensitive!**
Lembre-se que a comparação é case-sensitive:

```go
// ❌ Vai falhar
{Message: "PARAMETER 'b'", StartLine: 4}

// ✅ Vai passar  
{Message: "Parameter 'b'", StartLine: 4}
```

## 🚀 **Fluxo Recomendado**

1. **Crie o arquivo .clj** com casos de teste
2. **Execute manualmente** `./arit arquivo.clj` 
3. **Se não funcionar**, use `DebugRuleTest()`
4. **Copie as sugestões** geradas pelo debug
5. **Substitua por `RunRuleTest()`** no teste final
6. **Execute o teste** para confirmar que passa

## 📋 **Checklist de Debug**

- [ ] Arquivo .clj criado com casos representativos
- [ ] Execução manual testada
- [ ] Debug test executado (se necessário)  
- [ ] Mensagens copiadas corretamente
- [ ] Linhas verificadas (começam em 1)
- [ ] Teste final executado com sucesso
- [ ] `DebugRuleTest` removido/comentado

---

**Lembre-se**: O debug é uma ferramenta temporária. Use-o para descobrir as mensagens e depois volte para `RunRuleTest()` no código final! 🎯 