(ns teste.examples.refactored.verbose-checks-refactored
  "Exemplos de verificações idiomáticas - versão refatorada")

;; ============================================================================
;; COMPARAÇÕES NUMÉRICAS IDIOMÁTICAS
;; ============================================================================

(defn check-zero-idiomatic [x]
  ;; Versão idiomática usando zero?
  (zero? x))

(defn check-zero-idiomatic-reversed [x]
  ;; Versão idiomática usando zero?
  (zero? x))

(defn check-positive-idiomatic [x]
  ;; Versão idiomática usando pos?
  (pos? x))

(defn check-positive-idiomatic-reversed [x]
  ;; Versão idiomática usando neg? (lógica invertida)
  (neg? x))

(defn check-negative-idiomatic [x]
  ;; Versão idiomática usando neg?
  (neg? x))

(defn check-negative-idiomatic-reversed [x]
  ;; Versão idiomática usando pos? (lógica invertida)
  (pos? x))

;; ============================================================================
;; COMPARAÇÕES BOOLEANAS IDIOMÁTICAS
;; ============================================================================

(defn check-true-idiomatic [x]
  ;; Versão idiomática usando true?
  (true? x))

(defn check-true-idiomatic-reversed [x]
  ;; Versão idiomática usando true?
  (true? x))

(defn check-false-idiomatic [x]
  ;; Versão idiomática usando false?
  (false? x))

(defn check-false-idiomatic-reversed [x]
  ;; Versão idiomática usando false?
  (false? x))

;; ============================================================================
;; COMPARAÇÕES COM NIL IDIOMÁTICAS
;; ============================================================================

(defn check-nil-idiomatic [x]
  ;; Versão idiomática usando nil?
  (nil? x))

(defn check-nil-idiomatic-reversed [x]
  ;; Versão idiomática usando nil?
  (nil? x))

;; ============================================================================
;; OPERAÇÕES MATEMÁTICAS IDIOMÁTICAS
;; ============================================================================

(defn increment-idiomatic [x]
  ;; Versão idiomática usando inc
  (inc x))

(defn increment-idiomatic-reversed [x]
  ;; Versão idiomática usando inc
  (inc x))

(defn decrement-idiomatic [x]
  ;; Versão idiomática usando dec
  (dec x))

;; ============================================================================
;; EXEMPLOS EM CONTEXTOS MAIS COMPLEXOS - VERSÃO IDIOMÁTICA
;; ============================================================================

(defn complex-example-1-idiomatic [coll]
  ;; Versão idiomática
  (when (zero? (count coll))
    (println "Collection is empty")))

(defn complex-example-2-idiomatic [x y]
  ;; Versão idiomática
  (cond
    (true? x)
    "x is true"
    (nil? y)
    "y is nil"
    (pos? x)
    "x is positive"
    :else
    "default"))

(defn complex-example-3-idiomatic [numbers]
  ;; Versão idiomática usando inc diretamente
  (map inc numbers))

(defn complex-example-4-idiomatic [values]
  ;; Versão idiomática usando false? diretamente
  (filter false? values))

(defn complex-example-5-idiomatic [data]
  ;; Versão idiomática
  (let [is-zero (zero? data)
        is-nil  (nil? data)
        incremented (inc data)]
    [is-zero is-nil incremented]))

;; ============================================================================
;; CASOS EDGE - PERMANECEM INALTERADOS
;; ============================================================================

(defn edge-case-1-idiomatic [x]
  ;; Comparação com outros números (correto como está)
  (= 5 x))

(defn edge-case-2-idiomatic [x]
  ;; Operação com outros números (correto como está)
  (+ 2 x))

(defn edge-case-3-idiomatic [x]
  ;; Subtração de outros números (correto como está)
  (- x 5))

(defn edge-case-4-idiomatic [x y]
  ;; Comparação entre variáveis (correto como está)
  (= x y))

;; ============================================================================
;; EXEMPLOS ANINHADOS - VERSÃO IDIOMÁTICA
;; ============================================================================

(defn nested-example-1-idiomatic [coll]
  ;; Versão idiomática
  (if (zero? (count coll))
    (true? (empty? coll))
    (pos? (count coll))))

(defn nested-example-2-idiomatic [x]
  ;; Versão idiomática
  (inc (dec x)))

;; ============================================================================
;; EXEMPLOS COM THREADING MACROS - VERSÃO IDIOMÁTICA
;; ============================================================================

(defn threading-example-1-idiomatic [x]
  ;; Versão idiomática
  (-> x
      inc
      zero?))

(defn threading-example-2-idiomatic [coll]
  ;; Versão idiomática
  (->> coll
       (filter pos?)
       (map inc)))

;; ============================================================================
;; EXEMPLOS ADICIONAIS DE BOAS PRÁTICAS
;; ============================================================================

(defn good-practices-1 [numbers]
  ;; Usando funções idiomáticas em combinação
  (->> numbers
       (filter pos?)
       (map inc)
       (remove zero?)))

(defn good-practices-2 [data]
  ;; Verificações idiomáticas em condicionais
  (cond
    (nil? data) :nil
    (zero? data) :zero
    (pos? data) :positive
    (neg? data) :negative
    :else :unknown))

(defn good-practices-3 [coll]
  ;; Combinando verificações idiomáticas
  (and (not (nil? coll))
       (pos? (count coll))
       (not (zero? (first coll)))))

(defn good-practices-4 [x]
  ;; Usando funções idiomáticas em let
  (let [incremented (inc x)
        is-positive (pos? incremented)
        is-zero (zero? incremented)]
    {:value incremented
     :positive? is-positive
     :zero? is-zero}))

;; ============================================================================
;; DEMONSTRAÇÃO DE PERFORMANCE E CLAREZA
;; ============================================================================

(defn performance-example [numbers]
  ;; Versão idiomática é mais clara e potencialmente mais rápida
  (transduce
    (comp (filter pos?)
          (map inc)
          (remove zero?))
    conj
    []
    numbers))

(defn clarity-example [data-map]
  ;; Verificações idiomáticas tornam o código mais legível
  (-> data-map
      :value
      ((fn [x]
         (cond
           (nil? x) "No value"
           (zero? x) "Zero value"
           (pos? x) "Positive value"
           (neg? x) "Negative value"
           :else "Unknown value")))))

;; ============================================================================
;; EXEMPLOS DE COMPOSIÇÃO DE FUNÇÕES
;; ============================================================================

(defn composition-example-1 [numbers]
  ;; Composição usando funções idiomáticas
  (comp (partial filter pos?)
        (partial map inc)
        (partial remove zero?)))

(defn composition-example-2 [x]
  ;; Composição de verificações
  ((comp zero? inc dec) x))

;; ============================================================================
;; EXEMPLOS COM PREDICADOS CUSTOMIZADOS
;; ============================================================================

(defn custom-predicates [data]
  ;; Usando funções idiomáticas como base para predicados customizados
  (let [positive-and-not-zero? (every-pred pos? (complement zero?))
        nil-or-false? (some-fn nil? false?)]
    {:positive-non-zero (positive-and-not-zero? data)
     :nil-or-false (nil-or-false? data)}))

;; ============================================================================
;; EXEMPLOS DE VALIDAÇÃO
;; ============================================================================

(defn validation-example [input]
  ;; Validação usando funções idiomáticas
  (cond
    (nil? input) {:valid false :reason "Input cannot be nil"}
    (not (number? input)) {:valid false :reason "Input must be a number"}
    (zero? input) {:valid false :reason "Input cannot be zero"}
    (neg? input) {:valid false :reason "Input must be positive"}
    :else {:valid true :value (inc input)}))

;; ============================================================================
;; EXEMPLOS DE TRANSFORMAÇÃO DE DADOS
;; ============================================================================

(defn data-transformation-example [data-seq]
  ;; Transformação usando funções idiomáticas
  (->> data-seq
       (map (fn [item]
              (cond-> item
                (number? item) inc
                (nil? item) (constantly 0)
                (false? item) (constantly 1))))
       (filter pos?)
       (remove zero?)))

;; ============================================================================
;; EXEMPLOS DE AGREGAÇÃO
;; ============================================================================

(defn aggregation-example [numbers]
  ;; Agregação usando funções idiomáticas
  {:total-count (count numbers)
   :positive-count (count (filter pos? numbers))
   :zero-count (count (filter zero? numbers))
   :negative-count (count (filter neg? numbers))
   :nil-count (count (filter nil? numbers))
   :incremented-sum (reduce + (map inc numbers))}) 