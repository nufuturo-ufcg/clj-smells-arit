# Novos Exemplos de Code Smells

Este documento descreve os novos arquivos de exemplo criados para demonstrar dois code smells específicos implementados no projeto ARIT.

## 📁 Estrutura dos Arquivos

### Redundant Do Block

- **Smell**: `smells/redundant_do_block.clj`
- **Refatorado**: `refactored/redundant_do_block_refactored.clj`

### Positional Return Values

- **Smell**: `smells/positional_return_values.clj`
- **Refatorado**: `refactored/positional_return_values_refactored.clj`

## 🔍 Redundant Do Block

### Descrição
Identifica blocos `do` explícitos que são desnecessários porque suas formas pai já fornecem um `do` implícito.

### Exemplos Incluídos
O arquivo `redundant_do_block.clj` contém 15 exemplos diferentes de blocos `do` redundantes em:

1. **`when`** - `when` já fornece `do` implícito
2. **`let`** - `let` já fornece `do` implícito no corpo
3. **`if`** - Ramos do `if` com múltiplas expressões
4. **`defn`** - Corpo de função já tem `do` implícito
5. **`fn`** - Função anônima já tem `do` implícito
6. **`when-let`** - `when-let` já fornece `do` implícito
7. **`try/catch`** - Blocos `try` e `catch` já fornecem `do` implícito
8. **`cond`** - Ramos do `cond` (nota: `cond` NÃO fornece `do` implícito)
9. **`case`** - Ramos do `case` (nota: `case` NÃO fornece `do` implícito)
10. **`finally`** - Bloco `finally` já fornece `do` implícito
11. **`do` vazio** - Bloco `do` sem conteúdo
12. **`do` com única expressão** - Bloco `do` desnecessário
13. **Multi-arity functions** - Funções com múltiplas aridades
14. **`loop`** - `loop` já fornece `do` implícito
15. **`binding`** - `binding` já fornece `do` implícito

### Resultados dos Testes
- **Arquivo original**: 18 detecções de `redundant-do-block`
- **Arquivo refatorado**: 5 detecções restantes (casos onde `do` é realmente necessário)

## 📊 Positional Return Values

### Descrição
Detecta funções que retornam coleções sequenciais (vetores ou listas) onde o significado dos elementos é implícito por sua posição.

### Exemplos Incluídos
O arquivo `positional_return_values.clj` contém 20 exemplos diferentes:

1. **Informações de usuário** - Nome, idade, email, etc.
2. **Coordenadas geográficas** - Latitude, longitude
3. **Estatísticas de vendas** - Total, contagem, média, devoluções
4. **Resultado de operação** - Status, mensagem, ID, valor
5. **Configuração de banco** - Host, porta, database, usuário, senha
6. **Informações de arquivo** - Nome, tamanho, data, tipo MIME
7. **Resultado de validação** - Válido, erros, dados
8. **Dados de produto** - Nome, preço, categoria, estoque, disponível
9. **Informações de sessão** - ID, expiração, criação, lembrar
10. **Resultado de busca** - Resultados, total, tempo, tem mais
11. **Dados de performance** - Tempo, CPU, memória, status
12. **Informações de rede** - IP, máscara, gateway, DNS
13. **Análise de texto** - Palavras, sentenças, parágrafos, legibilidade
14. **Autenticação** - Autenticado, usuário, permissões, login
15. **Backup** - Nome arquivo, tamanho, criação, sucesso
16. **Total de pedido via `let`** - Subtotal, taxa, frete, total
17. **Cores RGB** - Valores RGB como lista literal
18. **Métricas do sistema** - CPU, memória, processos, status
19. **Processamento de imagem** - Largura, altura, formato, tamanho
20. **Transação financeira** - ID, valor, timestamp, status, taxa

### Resultados dos Testes
- **Arquivo original**: 25 detecções de `positional-return-values`
- **Arquivo refatorado**: 0 detecções (todos convertidos para mapas)

## 🔧 Refatorações Aplicadas

### Redundant Do Block
- Remoção de blocos `do` desnecessários
- Aproveitamento do `do` implícito das formas pai
- Casos especiais mantidos onde `do` é necessário (`if`, `cond`, `case`)

### Positional Return Values
- Conversão de vetores/listas para mapas com chaves descritivas
- Uso de keywords como chaves (`:name`, `:age`, `:email`, etc.)
- Melhoria na legibilidade e manutenibilidade do código

## 📈 Benefícios das Refatorações

### Redundant Do Block
- **Código mais limpo**: Remoção de sintaxe desnecessária
- **Melhor legibilidade**: Menos ruído visual
- **Idiomático**: Uso correto das construções Clojure

### Positional Return Values
- **Autodocumentação**: Chaves descritivas explicam o significado
- **Manutenibilidade**: Fácil adicionar/remover campos
- **Robustez**: Menos propenso a erros de ordem
- **Legibilidade**: Código mais expressivo e claro

## 🧪 Como Testar

```bash
# Testar detecção de code smells
./arit analyze teste/examples/smells/redundant_do_block.clj
./arit analyze teste/examples/smells/positional_return_values.clj

# Verificar refatorações
./arit analyze teste/examples/refactored/redundant_do_block_refactored.clj
./arit analyze teste/examples/refactored/positional_return_values_refactored.clj
```

## 📚 Referências

- [ClojureDocs - clojure.test](https://clojuredocs.org/clojure.test) - Framework de testes do Clojure
- [goclj](https://pkg.go.dev/github.com/cespare/goclj) - Ferramentas Go para trabalhar com código Clojure
- Documentação interna do projeto ARIT sobre code smells em Clojure 