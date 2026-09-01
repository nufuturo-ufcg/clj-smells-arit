# Contextual findings policy

The analyzer has two independent classifications:

- `proven`: the available Clojure syntax and semantic resolution are enough to
  establish the smell or semantic error without knowing the application's
  contract;
- `contextual`: the AST exposes a suspicious pattern, but deciding whether it
  is a smell requires information outside the file, such as ownership,
  lifecycle, cardinality, API contracts, or runtime configuration.

Only `proven` findings are shown by default. Contextual candidates are kept in
the analysis result and can be inspected with `--include-contextual`. Severity
does not determine classification: a `HINT` can be proven, and a warning can
be contextual.

## Absolute precision gate

False positives are not allowed. Every emitted finding, including a contextual
candidate, must correspond to a real and relevant occurrence of the structural
pattern described by its rule. Contextuality may express uncertainty about an
external contract, intent, lifecycle, ownership, or runtime behavior; it must
not be used to emit a speculative pattern that may not actually be present.

False negatives are an acceptable temporary cost of conservative analysis. When
the available evidence cannot establish that the occurrence is real, the rule
must remain silent or retain the case as an explicitly documented unsupported
variant. Recall improvements are accepted only after the zero-false-positive
gate remains satisfied.

Once that gate is satisfied, the optimization target is to prove as many
contextual findings as possible. A contextual-to-proven promotion is valid only
when new verifiable semantic evidence removes the uncertainty and the change
preserves every valid negative. Lower contextual volume is useful only when it
comes from justified promotions, never from suppressing evidence or weakening
the classification contract.

The file role (`production`, `test`, `fixture`, `generated`, `dev`, or
`build`) is recorded separately as `source_role`. A proven defect in a test or
fixture remains proven; source role may affect severity, but it must not turn
an objectively identifiable pattern into a contextual finding.

## Proven by local evidence

Examples include:

- a local/evaluated symbol used as a `case` test constant;
- `doall` wrapped around an already eager `mapv` or `filterv`;
- redundant `do` blocks with a single body expression;
- `when-not`/`if-not` directly around `empty?`, when the rewrite preserves
  truthiness;
- boolean or `nil` comparisons whose replacement preserves Clojure's value
  semantics.

## Contextual by design

Examples include general `doall`, numeric simplifications, collection
materialization and ownership, dynamic vars, namespace loading, broad imports,
channel lifecycle, direct runtime APIs, and style/architecture smells where a
valid intentional use cannot be ruled out from the file alone. The calibrated
`excessive-refers` rule is an explicit exception: once the resolved explicit
reference count reaches its empirically calibrated threshold, the statistical
outlier itself is proven even though the developer may still judge the design
to be intentional.

When a rule cannot prove the contract, it must create a contextual finding
with a reason. It must not use the file role, `HINT` severity, or a generic
smell label as a substitute for that evidence.

For `unnecessary-macros`, the only current promotion path is the explicit
`rule-config.unnecessary-macros.proven-macros` allow-list in an indexed
project. Promotion additionally requires at least one indexed call, valid
arity at every call, and proven evidence for every observed argument. The
default configuration remains contextual; unknown arguments, invalid arities,
missing index data, and callers outside the indexed corpus do not promote.

The semantic facts available to rules include symbol resolution, abstract
type, nilability, laziness, effects, execution phase, function summaries, and
optional project-index information. Synthetic catalog expectations are
recorded separately in `docs/CATALOG_SEMANTIC_EXPECTATIONS.yaml`; comments and
directory names are metadata and never trigger a finding.
