(ns multiple-evaluation-in-macros)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Basic double evaluation of macro argument in syntax quote
(defmacro double-eval-basic [expr]
  `(do
     (println "Executando:" ~expr)
     ~expr))

;; Example 2: Unhygienic macro duplicating expression in arithmetic operation
(defmacro square-unhygienic [x]
  `(* ~x ~x))

;; Example 3: Multiple evaluations inside a conditional 'when' block
(defmacro log-and-run [condition expr]
  `(when ~condition
     (println "Log:" ~expr)
     ~expr))

;; Example 4: Unhygienic repetition across let bindings without gensym shield
(defmacro bad-bindings [val-expr]
  `(let [a ~val-expr
         b ~val-expr]
     (+ a b)))

;; Example 5: Multiple evaluations inside loop constructs (doseq)
(defmacro repeat-eval-loop [coll-expr]
  `(doseq [x# ~coll-expr
           y# ~coll-expr]
     (println x# y#)))

;; Example 6: Argument evaluated in try block and re-evaluated in catch/finally
(defmacro try-re-eval [expr]
  `(try
     ~expr
     (catch Exception e#
       (println "Falhou ao avaliar:" ~expr))))

;; Example 7: Macro argument evaluated multiple times in map construction
(defmacro pair-value [expr]
  `{:raw ~expr
    :processed (str ~expr)})

;; Example 8: Multiple evaluation hidden via unquoted vector unpacking
(defmacro unpack-twice [items-expr]
  `(concat ~items-expr ~items-expr))

;; Example 9: Naive auto-gensym usage evaluating ~expr multiple times in bindings (False Negative)
(defmacro false-safe-gensym [expr]
  `(let [a# ~expr
         b# ~expr]
     (+ a# b#)))

;; Example 10: Triple evaluation of assertion expression in error messaging
(defmacro assert-verbose [expr]
  `(if ~expr
     (println "Sucesso com valor:" ~expr)
     (throw (Exception. (str "Falhou no teste da expressão: " ~expr)))))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Hygienic evaluation using local variable with auto-gensym (#)
(defmacro square-hygienic [x]
  `(let [x# ~x]
     (* x# x#)))

;; Example 12: Single evaluation bound to an explicit gensym variable (False Positive)
(defmacro safe-square-explicit-gensym [x]
  (let [g (gensym "val")]
    `(let [~g ~x]
       (* ~g ~g))))

;; Example 13: Single evaluation of argument in syntax quote
(defmacro single-eval-safe [expr]
  `(inc ~expr))

;; Example 14: Argument appears in mutually exclusive branches of if (False Positive)
(defmacro se-debug [flag expr fallback]
  `(if ~flag
     ~expr       ;; Branch A: Executes ONLY if flag is true
     ~fallback)) ;; Branch B: Executes ONLY if flag is false

;; Example 15: Argument evaluated once and quoted (') for metadata/logging (False Positive)
(defmacro trace-ast-safe [expr]
  `(do
     (println "AST estática:" '~expr) ;; Quoted: static data, 0 evaluations
     ~expr))                          ;; Single runtime evaluation

;; Example 16: Argument used in mutually exclusive cond branches (False Positive)
(defmacro mutually-exclusive-cond [test-expr fallback-expr]
  `(cond
     ~test-expr :ok
     :else ~fallback-expr))

;; Example 17: Macro with auto-gensym let binding wrapping multiple references safely
(defmacro safe-pair-value [expr]
  `(let [v# ~expr]
     {:raw v#
      :processed (str v#)}))

;; Example 18: Pure static transformation without runtime syntax-quote unquote
(defmacro static-macro-transformer [a b]
  (list 'def a b))

;; Example 19: Argument unquoted inside a non-evaluated comment block (False Positive)
(defmacro comment-unquote-safe [expr]
  `(do
     (comment (println ~expr))
     ~expr))

;; Example 20: Safe parking/deferral using let binding with auto-gensym inside go block
(defmacro safe-go-eval [expr]
  `(let [val# ~expr]
     (clojure.core.async/go
       (println val#)
       val#)))

;; Expansion-time tests do not evaluate the caller expression at runtime.
(defmacro expansion-branch-safe [expr]
  (if (string? expr)
    `(consume-string ~expr)
    `(consume-value ~expr)))

;; Occurrences in distinct arities must never be added together.
(defmacro separate-arities-safe
  ([expr] `(consume-one ~expr))
  ([expr fallback] `(consume-two ~expr ~fallback)))

;; The same argument in mutually exclusive runtime branches is evaluated once.
(defmacro same-arg-exclusive-safe [flag expr]
  `(if ~flag ~expr ~expr))

;; Unknown template transformations are suppressed in high-precision mode.
(defmacro unknown-template-wrapper-safe [expr]
  (with-meta `(+ ~expr ~expr) {:generated true}))

;; Repeating a variadic splice duplicates every caller body form.
(defmacro duplicate-body [& body]
  `(do ~@body ~@body))

;; Only the risky arity should be sufficient to report the macro once.
(defmacro one-risky-arity
  ([expr] `(consume ~expr))
  ([expr fallback] `(vector ~expr ~expr ~fallback)))

;; A generated function name is declarative; its single call-site is one use.
(defmacro generated-function-name-safe [fn-name]
  `(defn ~fn-name
     ([value#] (~fn-name value# nil))
     ([value# fallback#] (or value# fallback#))))

;; Generated function arities are alternative invocation paths, not a sequence.
(defmacro generated-arities-safe [expr]
  `(fn
     ([value#] (consume ~expr value#))
     ([value# fallback#] (consume ~expr value# fallback#))))
