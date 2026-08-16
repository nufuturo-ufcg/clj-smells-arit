# Backlog de melhoria da precisão do Arit

O acompanhamento consolidado com checkboxes, resultados concluídos e próximas fases está
em [`PLANO_ACOMPANHAMENTO_PRECISAO_ARIT.md`](PLANO_ACOMPANHAMENTO_PRECISAO_ARIT.md).

## Objetivo

Este documento registra as regras que ainda precisam ser calibradas após a análise do
`Common-Metadata-Repository` (CMR). O objetivo exclusivo é aumentar a precisão do Arit.
Quando não for possível provar um defeito ou uma transformação segura, a regra não deve
afirmá-los. Isso não autoriza inferir intenção: se o código contém um padrão observável
com risco documentado, o ARIT deve avisar e deixar ao desenvolvedor a avaliação do uso
deliberado. Falsos negativos continuam preferíveis a acusações incorretas, mas um aviso de
revisão precisamente redigido não é uma acusação de defeito.

Este backlog não propõe mudanças no CMR, nos resultados históricos ou nos scripts de
análise. As alterações devem ocorrer somente no analisador e em seus testes.

## Linha de base

A linha de base foi produzida com o executável de SHA-256
`0fc294e9d81172b9e1661d2be88296f0fd62bc0650a95e58e28bdc8e4a1ef337`.

- Total de achados no CMR: **665**.
- `nested-atoms` e `marker-protocol`: nenhum achado restante.
- `immutability-violation`: 2 achados de alta confiança.
- `blocking-inside-go`: 3 achados de alta confiança, incluindo 2 fluxos transitivos.
- Os números abaixo medem achados, não defeitos confirmados. Uma contagem alta não é
  evidência de que a regra esteja correta.

| Regra | Achados | Prioridade | Direção recomendada |
|---|---:|---|---|
| `improper-emptiness-check` | 101 | P1 | Restringir ainda mais o contexto da recomendação |
| `nested-forms` | 95 | P0 | Exigir equivalência estrutural; remover sugestões genéricas |
| `map-with-nil-values` | 74 | P0 | Redefinir o risco e calibrar a mensagem sem desabilitar |
| `verbose-checks` | 74 | P1 | Adicionar inferência de tipo e de contrato de retorno |
| `implicit-namespace-dependencies` | 72 | P0 | Tratar DSLs e exigir risco concreto de colisão |
| `unnecessary-into` | 67 | P0 | Validar equivalência de tipo, ordem e realização |
| `thread-ignorance` | 58 | P0 | Exigir cadeia linear, uso único e posição compatível |
| `production-doall` | 33 | P0 | Avisar sobre realização sem inferir intenção; separar risco de redundância |
| `relying-on-load-time-side-effects` | 16 | P1 | Distinguir recurso imutável de I/O externo mutável |
| `unnecessary-laziness` | 15 | P1 | Avisar materialização observável e separar risco de equivalência |
| `redundant-do-block` | 11 | P1 | Considerar macros, metadados e posições sintáticas |
| `unnecessary-macros` | 9 | P0 | Preservar macros de especialização e performance |
| `misuse-of-dynamic-scope` | 8 | P0 | Exigir uso problemático, não apenas definição ou binding |
| `case-with-non-literal-test-values` | 6 | P1 | Corrigir diagnóstico e modelar constantes válidas |
| `multiple-evaluation-in-macros` | 6 | Monitorar | Manter conjunto conservador atual |
| `misused-threading` | 5 | P0 | Não considerar função anônima um smell por si só |
| `excessive-refers` | 4 | Monitorar | Preservar limiar empírico e documentar sua reprodutibilidade |
| `blocking-inside-go` | 3 | Monitorar | Ampliar apenas com resolução interarquivo segura |
| `private-multimethods` | 3 | P0 | Remover a premissa de que privacidade é smell |
| `immutability-violation` | 2 | Monitorar | Manter os dois casos atuais como baseline positivo |
| `unmanaged-resource-io` | 2 | Monitorar | Confirmar ownership interprocedural antes de ampliar |
| `conditional-build-up` | 1 | Monitorar | Manter somente transformação local comprovável |

### Atualização posterior da linha de base — `nested-forms`

Em 2026-08-13, `nested-forms` foi restringida aos subtipos estruturalmente seguros
`nested-let-flattening` e `nested-doseq-flattening`, sem desabilitar a regra. Com o
executável SHA-256
`72f4f782c8ff55b4db80c904185d35fcb5419380eee023739027e826e91318ba`, a execução no CMR
produziu **3** achados da regra, contra **95** na baseline, e **573** achados totais.
Nenhuma outra contagem por regra mudou. A baseline original acima permanece registrada
para permitir comparação histórica.

Em seguida, `thread-ignorance` foi restringida a pipelines com funções resolvidas,
posição de argumento consistente e fluxo linear. Com o executável SHA-256
`064bc078883686b9f152eb90f1fa418568212bc6941f225bbe58dbb5abfd3e08`, a regra passou de
**58** achados para **0** no CMR e o total geral passou de **573** para **515**. A regra
permanece habilitada; a ausência de achados significa que nenhum dos casos reais satisfez
o novo contrato de alta confiança.

Por fim, `misused-threading` deixou de considerar mistura de domínios e presença de
função anônima como evidência. A semântica de posição foi compartilhada com
`thread-ignorance`, exigindo ao menos duas etapas resolvidas, todas na direção oposta à
macro. `into` foi excluída da inferência por possuir papéis legítimos de coleção tanto no
primeiro quanto no último argumento. Com o executável SHA-256
`756e49ce800ede2dd9d844ccef34ed4a68f98294f5d87dc855738241861c22fc`, os **5** falsos
positivos anteriores e **3** candidatos intermediários com `into` foram eliminados; a
regra permaneceu habilitada e terminou com **0 achados** no CMR.

Na etapa seguinte, `unnecessary-into` foi restringida à fusão exata de uma operação lazy
transducer-capable diretamente em `(into [] ...)`. Conversões genéricas, mapas, sets,
destinos dinâmicos, `mapv`, `pmap`, múltiplas fontes e formas não avaliadas passaram a ser
negativas. Com o executável SHA-256
`b0ec49495345a150a9b6a3822b3adbdbeb81dc47e0d684f3e4ec41cd68ad8fe5`, a regra passou
de **67** para **8 achados** no CMR; os oito foram revisados como fusões seguras e nenhuma
outra contagem por regra mudou.

`production-doall` inicialmente foi restringida a dupla realização comprovada, mas essa
decisão foi corrigida para não usar intenção presumida como supressão. Com o executável
SHA-256 `27b9295d6e2a6ca13a0d0501de0065ef090a4f80bb7c247ebdb1069b17e23a7b`, a regra
produz **35 achados** no CMR: **3** diagnósticos de redundância para `(doall (mapv ...))`
e **32** avisos de realização/retenção que exigem avaliação humana. Esses 32 avisos não
afirmam que o código está errado; também não são descartados como “intencionais” pelo
analisador.

## Melhorias transversais obrigatórias

### 1. Classificador único de código de teste e desenvolvimento — P0

O filtro atual reconhece diretórios como `test/` e `tests/`, mas deixa passar estruturas
comuns do CMR, incluindo `int-test/`, `dev-system/` e `sample-data/`. Isso afeta qualquer
regra que trate código de produção de forma diferente de fixtures, ferramentas ou testes.

Alteração proposta:

- Criar uma classificação central de arquivo: `production`, `test`, `integration-test`,
  `dev`, `generated` e `sample`.
- Reconhecer componentes completos do caminho, e não apenas buscas por substring.
- Permitir configuração por projeto, sem colocar exceções específicas do CMR nas regras.
- Expor a classificação no contexto do analisador para todas as regras.
- Não suprimir testes quando o usuário habilitar explicitamente `analyze-tests`.

Critério de aceite:

- Fixtures em `int-test/` não podem ser apresentadas como problemas de produção.
- Um arquivo cujo nome apenas contenha a palavra `test` não deve ser classificado
  incorretamente.
- A mesma lógica deve ser usada pelo carregamento de arquivos e pelas regras.

### 2. Níveis explícitos de confiança — P0

Hoje, severidade e confiança estão misturadas. Uma sugestão estética pode aparecer ao
lado de um defeito semanticamente comprovado.

Alteração proposta:

- Adicionar confiança `proven`, `high`, `heuristic` ao achado ou à regra.
- Usar confiança para calibrar a linguagem e a ordenação, nunca para presumir intenção ou
  desabilitar uma regra.
- Habilitar por padrão somente achados `proven` e `high`.
- Tornar regras puramente heurísticas opt-in, mesmo quando sua severidade for `HINT`.
- Exigir que cada achado registre a evidência usada: resolução do símbolo, tipo inferido,
  posição sintática e contexto de execução.

Critério de aceite:

- Uma recomendação baseada apenas em limiar de tamanho não pode ser confundida com um
  defeito semântico.
- O modo padrão deve privilegiar precisão; um modo exploratório pode recuperar as
  heurísticas.

### 3. Prova de equivalência para sugestões de reescrita — P0

As regras de estilo frequentemente sugerem código que parece equivalente, mas pode mudar
tipo de retorno, ordem, cardinalidade, avaliação lazy, exceções ou efeitos colaterais.

Alteração proposta:

- Centralizar verificações de equivalência em utilitários compartilhados.
- Validar tipo de retorno, posição do argumento, número de avaliações e ordem.
- Não sugerir uma reescrita quando qualquer uma dessas propriedades for desconhecida.
- Separar “há um problema” de “esta transformação específica é segura”.

## Regras P0

### `map-with-nil-values` — 74 achados

Problema atual:

- A regra assume que ausência de chave e chave presente com valor `nil` são equivalentes.
- Em Clojure, essas representações são observavelmente diferentes por `contains?`, merge,
  serialização, schemas, valores default e APIs externas.
- Os achados incluem mapas de configuração, documentos e estruturas com formato fixo.

Alteração proposta:

- Manter a regra habilitada, mas substituir a alegação de defeito por um aviso de revisão
  enquanto não houver evidência de contrato.
- Nunca sinalizar um literal ou `assoc` apenas porque o valor é `nil`.
- Uma futura versão só deve emitir quando o próprio fluxo demonstra que ausência e `nil`
  são tratados como equivalentes, ou quando um schema conhecido proíbe explicitamente
  `nil` e permite a ausência da chave.
- Tratar serialização, records, schemas e payloads externos como barreiras conservadoras.

Critério de aceite:

- Nenhum achado pode depender apenas da presença sintática de `nil`.
- Devem existir testes negativos para `contains?`, destructuring com default, JSON e merge.

### `nested-forms` — 95 achados

Estado: **implementado e validado**. A regra permanece habilitada; 3 achados seguros
restaram no CMR.

Problema atual:

- A profundidade é usada como aproximação de complexidade sem considerar semântica.
- Sugestões como `some->`, `cond`, fusão de `doseq` ou fusão de `let` nem sempre preservam
  controle de fluxo, tratamento de exceção, dependências entre bindings ou efeitos.
- Há sobreposição com `thread-ignorance`, `conditional-build-up` e `misused-threading`.

Alteração proposta:

- Remover recomendações genéricas geradas apenas pela profundidade.
- Detectar somente padrões locais para os quais exista reescrita comprovadamente
  equivalente.
- Não atravessar `try`, `catch`, `finally`, macros desconhecidas, loops ou funções.
- Exigir dependência linear antes de sugerir `some->`.
- Consolidar a taxonomia com as demais regras estruturais para evitar achados duplicados.

Critério de aceite:

- Cada subtipo da regra deve possuir transformação formalmente descrita e testes de
  equivalência positivos e negativos.
- Na ausência de uma transformação segura, a regra deve permanecer silenciosa.

### `thread-ignorance` — 58 achados

Estado: **implementado e validado**. A regra permanece habilitada e não houve achado no
CMR que satisfizesse todas as pré-condições de equivalência.

Problema atual:

- Bindings temporários podem documentar etapas, permitir inspeção, controlar avaliação ou
  ser reutilizados. Sua presença não prova que threading seja superior.
- Chamadas aninhadas não determinam automaticamente se `->` ou `->>` recebe o valor na
  posição correta.

Alteração proposta:

- Exigir uma cadeia linear com pelo menos três etapas.
- Cada binding intermediário deve ter uso único, posterior e direto.
- Resolver a função de cada etapa e provar a posição do argumento encadeado.
- Rejeitar cadeias com destructuring, branching, reutilização, efeitos colaterais,
  type hints, macros ou chamadas não resolvidas.
- Não emitir quando os nomes dos bindings carregarem informação de domínio relevante.

Critério de aceite:

- A transformação sugerida deve preservar ordem e quantidade de avaliações.
- Casos com valor usado duas vezes ou fora da posição de threading devem ser negativos.

### `unnecessary-into` — 8 achados atuais (67 na baseline)

Estado: **parcial e reaberto após revisão semântica**. A regra permanece habilitada e
cobre apenas fusão transducer em destino literal `[]`, mas ainda não prova que a origem
possui comportamento equivalente entre `Seqable` e `IReduceInit`.

Problema atual:

- A regra mistura dois smells: transformação de coleção e uso de transducer.
- Algumas mensagens sugerem transducer sobre operações já eager, como `mapv`.
- Substituições podem mudar tipo concreto, metadados, comportamento de duplicatas,
  realização e desempenho.

Alteração proposta:

- Separar `redundant-collection-conversion` de `transducer-opportunity`.
- Para conversão redundante, exigir que tipo de entrada e alvo sejam conhecidos.
- Para transducer, aceitar somente pipeline composto por operações transducer-capable,
  sem materialização intermediária e com consumidor compatível.
- Não recomendar transducer quando a entrada já for produzida por `mapv`, `filterv` ou
  outra operação eager.
- Validar preservação de ordem e semântica de chaves duplicadas para mapas.

Critério de aceite:

- [x] Toda mensagem contém uma expressão Clojure sintaticamente válida por aridade.
- [ ] Provar que o protocolo de percurso da origem preserva resultado, ordem,
  cardinalidade e realização.
- [x] Operadores eager, paralelos, sombreados ou com múltiplas fontes são negativos.
- [x] Conversões genéricas e destinos com semântica de colisão desconhecida são negativos.
- [x] Há mais casos negativos do que positivos nos testes.
- [x] Os oito sobreviventes do CMR foram revisados manualmente e reclassificados como
  oportunidades prováveis, não equivalências comprovadas.
- [ ] Rejeitar `ResolutionUnresolved`, usar o contexto de execução central e corrigir as
  opções de configuração inertes.
- [ ] Adicionar teste executável com `Seqable` e `IReduceInit` divergentes.

### `production-doall` — 35 achados atuais (33 na baseline)

Estado: **implementado e validado**. A regra permanece habilitada, separando dupla
realização demonstrável de aviso de risco observável.

Problema atual:

- `doall` força toda a realização e mantém a sequência realizada alcançável pelo retorno.
- A análise pode provar redundância em alguns produtores eager, mas não consegue determinar
  se realização, retenção, efeitos ou sincronização foram intencionais.
- Suprimir porque o uso parece comum em I/O, cache, transação ou paralelismo seria inferir
  intenção a partir do contexto.

Alteração proposta:

- Emitir para toda chamada avaliada e canonicamente resolvida de `clojure.core/doall`.
- Emitir diagnóstico específico de redundância somente para produtor eager conhecido, hoje
  `mapv` e `filterv` diretos.
- Nos demais casos, descrever realização e retenção como fatos e solicitar revisão de
  cardinalidade e ciclo de vida, sem recomendar remoção automática.
- Reconhecer efeitos, `with-open`, sincronização, batches e caches como contexto informativo
  futuro, nunca como prova de intenção ou motivo automático de supressão.

Critério de aceite:

- [x] I/O, efeitos, retorno materializado, `pmap`, sincronização e produtor desconhecido
  recebem aviso de revisão, sem alegação de redundância.
- [x] O achado explica que `mapv`/`filterv` já realizou a origem.
- [x] Shadowing, formas não avaliadas e aridade parcial são cobertos.
- [x] A mensagem declara que análise estática não infere intenção e delega a decisão ao
  desenvolvedor.
- [x] Os 35 achados do CMR foram enumerados: 3 redundâncias e 32 revisões.
- [ ] Adicionar confiança ao modelo serializado para que consumidores distingam os dois
  diagnósticos sem interpretar a mensagem.

### `implicit-namespace-dependencies` — 72 achados

Problema atual:

- `:refer :all` e `use` aumentam o espaço de símbolos, mas algumas bibliotecas de DSL,
  especialmente macros de roteamento, são tradicionalmente consumidas dessa forma.
- A regra relata risco abstrato sem provar ambiguidade ou colisão.

Alteração proposta:

- Tratar bibliotecas de DSL por configuração, não por exceções codificadas para o CMR.
- Exigir colisão real, shadowing ou uso ambíguo antes de emitir no perfil de alta precisão.
- Quando for possível, listar apenas os símbolos efetivamente usados como correção.
- Separar `use`, `:refer :all` e refer explícito excessivo em diagnósticos distintos.

Critério de aceite:

- Uma dependência ampla sem colisão não deve gerar warning no perfil padrão.
- O diagnóstico deve identificar os símbolos responsáveis pelo risco concreto.

### `private-multimethods` — 3 achados

Problema atual:

- Os três casos do CMR são `defmulti ^:private` usados para encapsular implementação.
- Privacidade de um multimétodo é uma escolha válida de API; não constitui smell por si só.
- `defmethod` precisa alcançar o var privado dentro do mesmo namespace, o que é legítimo.

Alteração proposta:

- Manter a regra habilitada e redefinir o diagnóstico atual para não afirmar que
  privacidade é defeito por si só.
- Se houver um smell real desejado, redefini-lo em termos de extensão externa comprovada
  de um multimétodo privado, e não da declaração privada em si.

Critério de aceite:

- `defmulti ^:private` com métodos no mesmo namespace deve ser negativo.

### `unnecessary-macros` — 9 achados

Problema atual:

- Todos os achados observados são wrappers matemáticos em um namespace que ativa
  `primitive-math` e chama métodos estáticos especializados.
- Transformá-los em funções pode introduzir boxing, alterar inferência primitiva e mudar o
  objetivo de expansão em tempo de compilação.

Alteração proposta:

- Reconhecer interop estático, type hints, bibliotecas de operadores primitivos, inline e
  contextos de especialização.
- Exigir que o corpo seja function-like e que não exista benefício observável de expansão.
- Suprimir quando a expansão preserva primitivos ou evita dispatch/boxing.
- Considerar a regra opt-in até que essa prova esteja disponível.

Critério de aceite:

- Wrappers de `StrictFastMath` sob `primitive-math` devem ser negativos.
- Um macro trivial puramente sintático continua positivo.

### `misuse-of-dynamic-scope` — 8 achados

Problema atual:

- Definir uma var dinâmica não prova mau uso.
- Três achados estão em fixtures de integração.
- Variáveis como `*request-id*` e `*read-eval*` têm usos legítimos conhecidos.
- A mensagem afirma perda em fronteira assíncrona mesmo quando a regra não localizou uma
  fronteira nem uma leitura da variável depois dela.

Alteração proposta:

- Remover achados baseados somente em `def ^:dynamic`.
- Para `binding`, exigir uma fronteira assíncrona real no corpo e uma leitura resolvida da
  var no fluxo delegado.
- Modelar quais construtores propagam bindings e quais não propagam.
- Usar o classificador de arquivos para fixtures e código dev.

Critério de aceite:

- Binding local sem travessia de thread deve ser negativo.
- A mensagem deve nomear a fronteira e o uso que perde contexto.

### `misused-threading` — 0 achados atuais (5 na baseline)

Estado: **implementado e validado**. A regra permanece habilitada e não houve achado no
CMR que satisfizesse o contrato posicional conservador.

Problema atual:

- Uma função anônima dentro de `->` ou `->>` pode ser a maneira mais clara e correta de
  posicionar o argumento.
- A existência de `#()` não prova perda de legibilidade nem erro de posição.

Alteração proposta:

- Remover o predicado “contém função anônima”.
- Emitir somente quando a expansão do threading coloca comprovadamente o valor em posição
  incompatível ou quando existe uma simplificação exata sem lambda.
- Comparar a forma expandida antes e depois da recomendação.

Critério de aceite:

- [x] Lambdas necessárias para posição arbitrária são negativas.
- [x] Mudança de tipo e heterogeneidade não geram recomendação estética.
- [x] Aliases, shadowing, quote, `some->`/`some->>` e aridade foram cobertos.
- [x] Funções com posição ambígua, especialmente `into`, permanecem silenciosas.
- [x] Há mais cenários negativos do que positivos nos testes.

## Regras P1

### `relying-on-load-time-side-effects` — 16 achados

Todos os achados restantes envolvem `slurp` em tempo de carga. Nem todo `slurp` desse tipo
tem o mesmo risco: carregar um recurso imutável empacotado é diferente de ler um caminho
externo mutável.

Melhoria requerida:

- Resolver a origem do argumento de `slurp`.
- Suprimir recursos classpath imutáveis e dados explicitamente definidos como constantes,
  salvo se houver custo ou falha relevante comprovável.
- Priorizar filesystem externo, rede, configuração mutável e inicialização repetida.
- Reconhecer `delay`, funções de startup e lifecycle como alternativas já seguras.

### `unnecessary-laziness` — 15 achados atuais

A realização imediata por `vec` é observável, mas nem toda operação possui equivalente
eager direto. `distinct`, `remove`, `keep` e `take` exigem contratos e reescritas
dependentes da origem.

Estado implementado:

- Resolução canônica de `vec` e do produtor lazy, sem fallback textual.
- Aviso de revisão para o padrão observado, sem presumir intenção ou afirmar defeito.
- Mensagem explicita tipo, ordem, cardinalidade, efeitos, chunking e entradas infinitas.
- Shadowing, aliases desconhecidos, formas não avaliadas e produtores eager são negativos.

Pendência restante:

- Auditar manualmente os 15 findings do CMR.
- Adicionar contratos de tipo/protocolo caso seja necessário elevar algum caso a
  equivalência comprovada; até lá, não recomendar reescrita automática.

### `verbose-checks` — 74 achados

A calibração anterior eliminou comparações de igualdade com zero sem prova de inteiro e
formas booleanas não equivalentes. Ainda falta:

- Propagar type hints e retornos conhecidos entre funções locais.
- Distinguir código de API, no qual retorno booleano estrito pode ser contrato.
- Validar equivalência numérica para ratios, BigDecimal, NaN e tipos Java.
- Suprimir recomendações quando a forma atual comunica melhor um protocolo externo.

### `improper-emptiness-check` — 101 achados

As sugestões de `seq` agora estão limitadas a posições de truthiness, mas a regra ainda é
majoritariamente estilística.

Melhoria requerida:

- Distinguir coleções counted de sequências lazy; o benefício não é igual.
- Evitar recomendação em macros desconhecidas e predicados que deliberadamente retornam
  booleano estrito.
- Reconhecer quando `count` já foi calculado e reutilizado, evitando sugerir nova travessia.
- Considerar deixar a regra opt-in como `HINT` de estilo.

### `case-with-non-literal-test-values` — 6 achados

Os exemplos com `(or ...)` entre constantes de `case` são defeitos reais prováveis, pois
`case` não avalia essa expressão. Entretanto, o diagnóstico atualmente imprime uma
expressão vazia.

Melhoria requerida:

- Reconstruir corretamente o test constant na mensagem e na localização.
- Modelar agrupamentos literais válidos, reader conditionals e metadados.
- Diferenciar lista literal intencional de tentativa de executar uma forma.
- Oferecer correção concreta, como repetir os pares ou trocar por `cond`, apenas quando
  semanticamente segura.

### `redundant-do-block` — 11 achados

Os casos em `try`, `catch` e funções parecem fortes, mas a remoção pode afetar metadados,
macros que recebem formas ou blocos usados como marcadores.

Melhoria requerida:

- Exigir que o pai seja uma forma core resolvida com corpo implicitamente sequencial.
- Não atravessar macros desconhecidas.
- Preservar metadados e comentários associados ao `do`.
- Separar `do` vazio de `do` com uma única expressão.

### `excessive-refers` — 0 achados atuais (4 na baseline)

O limiar não é arbitrário. Ele foi calculado como **média + 2 desvios-padrão** em uma
amostra de **430 repositórios**. O valor operacional implementado é 24 referências
explícitas por namespace, com comparação inclusiva (`>= 24`). A regra mede um outlier
estatístico; não prova colisão nem intenção.

Trabalho restante:

- Registrar os valores exatos de média e desvio-padrão, a política de arredondamento e o
  manifesto/versão dos 430 repositórios.
- Confirmar que a coleta original somava todas as referências por namespace, como faz a
  implementação atual.
- Manter o aviso baseado em quantidade, explicando que se trata de outlier estatístico e
  deixando a avaliação do uso ao desenvolvedor.
- Tratar colisões concretas, símbolos não usados ou perda de origem como evidências
  adicionais, sem torná-las condição para o aviso estatístico.

Validação atual: regressão 23/24 e suíte completa aprovadas; CMR com 0 achados da regra e
449 achados totais no executável
`bcf1f7b4be117438dada484ab71fa20f4da32a87d717ce2468cc57665ae74186`.

## Regras que devem permanecer em monitoramento

Estas regras não são prioridade de alteração enquanto os achados atuais continuarem
confirmados por inspeção:

- `blocking-inside-go`: 3 casos; `Thread/sleep`, leitura bloqueante transitiva e HTTP
  transitivo dentro de `go`/`go-loop`.
- `immutability-violation`: 2 usos de `def` dentro de função.
- `multiple-evaluation-in-macros`: 6 macros com inserção repetida em caminhos executáveis.
- `unmanaged-resource-io`: 2 recursos sem fechamento comprovado. Antes de ampliar, é
  necessário modelar transferência de ownership entre funções.
- `conditional-build-up`: 1 cadeia local de atualizações condicionais. Não ampliar para
  padrões com efeitos ou branches heterogêneos.

O monitoramento deve ser reaberto se uma dessas regras crescer abruptamente em outro
repositório ou se aparecer um contraexemplo real.

## Ordem recomendada de implementação

1. Implementar classificador de arquivos e níveis de confiança.
2. Redefinir `map-with-nil-values` e `private-multimethods`, mantendo ambas habilitadas.
3. Corrigir o grupo estrutural: `nested-forms`, `thread-ignorance` e `misused-threading`.
4. Separar e restringir `unnecessary-into`, `production-doall` e
   `unnecessary-laziness`.
5. Refinar `implicit-namespace-dependencies`, `misuse-of-dynamic-scope` e
   `relying-on-load-time-side-effects`.
6. Corrigir diagnóstico de `case-with-non-literal-test-values`.
7. Reavaliar `verbose-checks`, `improper-emptiness-check`, `redundant-do-block` e
   `excessive-refers` como regras opt-in de estilo.

## Processo de validação para cada regra

Cada alteração deve cumprir todos os passos abaixo:

1. Adicionar casos positivos mínimos com o motivo semântico explícito.
2. Adicionar mais casos negativos do que positivos, cobrindo shadowing, aliases, macros,
   quoting, execução diferida, tipos diferentes e caminhos de teste/dev.
3. Executar a suíte completa com `go test ./... -count=1`.
4. Executar os exemplos sintéticos sem alterar os arquivos do catálogo.
5. Executar o Arit no CMR e revisar manualmente todo achado novo da regra modificada.
6. Comparar fingerprints antes/depois; nenhum achado novo deve ser aceito somente porque
   aumentou recall.
7. Documentar falsos negativos deliberados quando a informação estática for insuficiente.

## Condição de encerramento

Uma regra está suficientemente calibrada quando:

- todo achado contém evidência verificável e mensagem específica;
- não depende apenas de preferência estética ou limiar arbitrário;
- a correção sugerida preserva semântica comprovadamente;
- aliases, shadowing, imports, macros e contexto de execução são respeitados;
- o contexto de teste/dev pode ser informado, mas não usado para deduzir intenção e
  suprimir o padrão;
- a revisão no repositório real não encontra falso positivo conhecido.

Se esses critérios não puderem ser satisfeitos, a regra deve permanecer habilitada com
linguagem de risco/revisão proporcional à evidência, sem alegar defeito ou correção segura
que a análise não consiga provar.
