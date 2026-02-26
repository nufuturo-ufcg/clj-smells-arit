# Grafo de dependências: como usar e gerar todos os tipos

O comando **`arit deps`** constrói o grafo unificado de dependências a partir dos seus arquivos Clojure e pode exportá-lo em **HTML** (grafo interativo). Este documento explica como usar o comando e como gerar cada tipo de análise.

---

## Uso básico

```bash
./arit deps <arquivo-ou-diretório> [arquivo-ou-diretório ...] [flags]
```

- **Sem flags de análise** e **sem `-o`**: o grafo completo é exibido em HTML (comportamento padrão quando não há análise).
- **Com flag de análise** (ex.: `--entries`) e **sem `-o`**: imprime apenas a **lista de IDs** no terminal.
- **Com `-o`** (ou `--output`): gera o **grafo em HTML** (completo ou do subgrafo da análise). Use `> arquivo.html` para salvar.

Exemplos rápidos:

```bash
# Grafo completo em HTML (salvar em arquivo)
./arit deps -o examples/deps/ > grafo.html

# Lista de entry points (só texto)
./arit deps --entries examples/deps/

# Grafo só dos entry points e do que eles chamam (HTML)
./arit deps --entries -o examples/deps/ > entries.html
```

---

## Tipos de análise e como gerar cada um

### 1. Grafo completo

Mostra todos os nós e todas as arestas do projeto analisado.

```bash
./arit deps -o examples/deps/ > full.html
```

---

### 2. Subgrafo a partir de um nó (`--from`)

Mostra apenas o nó indicado e tudo que é **alcançável** a partir dele seguindo arestas de chamada. O nó informado aparece destacado (focal) no HTML.

```bash
./arit deps --from myapp.core/run -o examples/deps/ > from_run.html
```

---

### 3. Nós alcançáveis a partir de um ID (`--reachable-from`)

- **Sem `-o`**: lista no terminal os IDs alcançáveis a partir do nó (seguindo arestas `calls`).
- **Com `-o`**: grafo em HTML só com esse subgrafo; o nó de partida fica destacado.

```bash
./arit deps --reachable-from myapp.core/main examples/deps/          # lista
./arit deps --reachable-from myapp.core/main -o examples/deps/ > reachable.html
```

---

### 4. Nós que alcançam um ID (`--reaching-to`)

Quem chama (direta ou indiretamente) o nó informado. O nó alvo fica destacado no HTML.

```bash
./arit deps --reaching-to myapp.db/query examples/deps/              # lista
./arit deps --reaching-to myapp.db/query -o examples/deps/ > reaching_to.html
```

---

### 5. Impacto ao mudar um nó (`--impact`)

Lista ou grafo dos **callers** do nó (quem é impactado se você mudar esse nó). O nó analisado fica destacado.

```bash
./arit deps --impact myapp.db/query examples/deps/                   # lista
./arit deps --impact myapp.db/query -o examples/deps/ > impact.html
```

---

### 6. Entry points e o que eles chamam (`--entries`)

Funções que **ninguém chama** (candidatas a entrada do programa) e tudo que elas chamam. No HTML, os entry points aparecem destacados.

```bash
./arit deps --entries examples/deps/                                # lista
./arit deps --entries -o examples/deps/ > entries.html
```

---

### 7. Código morto (`--dead-code`)

Funções **não alcançáveis** a partir de nenhum entry point. No HTML, essas funções aparecem destacadas. Se não houver código morto, o grafo completo é exibido com a mensagem "No dead code found".

```bash
./arit deps --dead-code examples/deps/                               # lista
./arit deps --dead-code -o examples/deps/ > dead_code.html
```

Para considerar apenas certas entradas (em vez de detectar automaticamente):

```bash
./arit deps --dead-code --dead-code-entries myapp.core/run,myapp.db/connect! -o examples/deps/ > dead_code_custom.html
```

---

### 8. Maior cadeia de chamadas (`--longest-path`)

A maior sequência de chamadas entre funções. O grafo em HTML mostra só essa cadeia; todos os nós do caminho ficam destacados.

```bash
./arit deps --longest-path examples/deps/                           # lista
./arit deps --longest-path -o examples/deps/ > longest_path.html
```

---

### 9. Camadas de namespaces (`--layers`)

**Só lista** (não gera grafo): imprime cada namespace e sua camada (0 = base, sem requires internos). Não usa `-o`.

```bash
./arit deps --layers examples/deps/
```

---

## Filtrar por tipo de aresta (`--edges`)

Você pode restringir o grafo a um ou mais **tipos de aresta**: `requires`, `refers`, `calls`, `reads`. Pode ser combinado com qualquer outra opção (ex.: `--from`, `--entries`).

```bash
# Só arestas de chamadas
./arit deps --edges calls -o examples/deps/ > only_calls.html

# Só requires e calls
./arit deps --edges requires,calls -o examples/deps/ > req_calls.html

# Subgrafo a partir de run, só arestas calls
./arit deps --from myapp.core/run --edges calls -o examples/deps/ > from_run_calls.html
```

Tipos válidos: **requires**, **refers**, **calls**, **reads** (separados por vírgula).

---

## Lista em vez de grafo (`--list`)

Com uma flag de análise, usar **`-o`** gera o grafo em HTML. Se quiser **forçar** só a lista de IDs no terminal, use **`--list`**:

```bash
./arit deps --reaching-to myapp.db/query --list examples/deps/
```

---

## Resumo rápido (comandos para gerar cada tipo em HTML)

| Objetivo                    | Comando (salvar em HTML) |
|----------------------------|---------------------------|
| Grafo completo             | `./arit deps -o examples/deps/ > full.html` |
| A partir de um nó          | `./arit deps --from myapp.core/run -o examples/deps/ > from_run.html` |
| Alcançáveis a partir de ID | `./arit deps --reachable-from myapp.core/main -o examples/deps/ > reachable.html` |
| Quem alcança um ID         | `./arit deps --reaching-to myapp.db/query -o examples/deps/ > reaching_to.html` |
| Impacto de um nó           | `./arit deps --impact myapp.db/query -o examples/deps/ > impact.html` |
| Entry points               | `./arit deps --entries -o examples/deps/ > entries.html` |
| Código morto               | `./arit deps --dead-code -o examples/deps/ > dead_code.html` |
| Maior cadeia de chamadas   | `./arit deps --longest-path -o examples/deps/ > longest_path.html` |
| Só arestas calls           | `./arit deps --edges calls -o examples/deps/ > only_calls.html` |
| Só requires e calls        | `./arit deps --edges requires,calls -o examples/deps/ > req_calls.html` |

---

## Documentação relacionada

- **[dependency-graph.md](dependency-graph.md)** — O que é o grafo unificado, estrutura (nós, arestas) e conceitos.
- **[graph-how-to-build.md](graph-how-to-build.md)** — Como o grafo é construído (duas fases, AST, mapeamento no código).
- **[graph-ideas.md](graph-ideas.md)** — Ideias de análises e visualizações em cima do grafo.
