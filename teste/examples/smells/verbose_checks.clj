(ns teste.examples.smells.verbose-checks
  "Exemplos de verificações verbosas que podem ser simplificadas")

;; ============================================================================
;; COMPARAÇÕES NUMÉRICAS VERBOSAS
;; ============================================================================

(defn check-zero-verbose [x]
  ;; Deve sugerir (zero? x)
  (= 0 x))

(defn check-zero-verbose-reversed [x]
  ;; Deve sugerir (zero? x)
  (= x 0))

(defn check-positive-verbose [x]
  ;; Deve sugerir (pos? x)
  (> x 0))

(defn check-positive-verbose-reversed [x]
  ;; Deve sugerir (neg? x) - inverte a lógica
  (> 0 x))

(defn check-negative-verbose [x]
  ;; Deve sugerir (neg? x)
  (< x 0))

(defn check-negative-verbose-reversed [x]
  ;; Deve sugerir (pos? x) - inverte a lógica
  (< 0 x))

;; ============================================================================
;; COMPARAÇÕES BOOLEANAS VERBOSAS
;; ============================================================================

(defn check-true-verbose [x]
  ;; Deve sugerir (true? x)
  (= true x))

(defn check-true-verbose-reversed [x]
  ;; Deve sugerir (true? x)
  (= x true))

(defn check-false-verbose [x]
  ;; Deve sugerir (false? x)
  (= false x))

(defn check-false-verbose-reversed [x]
  ;; Deve sugerir (false? x)
  (= x false))

;; ============================================================================
;; COMPARAÇÕES COM NIL VERBOSAS
;; ============================================================================

(defn check-nil-verbose [x]
  ;; Deve sugerir (nil? x)
  (= nil x))

(defn check-nil-verbose-reversed [x]
  ;; Deve sugerir (nil? x)
  (= x nil))

;; ============================================================================
;; OPERAÇÕES MATEMÁTICAS VERBOSAS
;; ============================================================================

(defn increment-verbose [x]
  ;; Deve sugerir (inc x)
  (+ 1 x))

(defn increment-verbose-reversed [x]
  ;; Deve sugerir (inc x)
  (+ x 1))

(defn decrement-verbose [x]
  ;; Deve sugerir (dec x)
  (- x 1))

;; ============================================================================
;; EXEMPLOS EM CONTEXTOS MAIS COMPLEXOS
;; ============================================================================

(defn complex-example-1 [coll]
  ;; Múltiplas verificações verbosas
  (when (= 0 (count coll))  ; Deve sugerir (zero? (count coll))
    (println "Collection is empty")))

(defn complex-example-2 [x y]
  ;; Verificações verbosas em condicionais
  (cond
    (= true x)  ; Deve sugerir (true? x)
    "x is true"
    (= nil y)   ; Deve sugerir (nil? y)
    "y is nil"
    (> x 0)     ; Deve sugerir (pos? x)
    "x is positive"
    :else
    "default"))

(defn complex-example-3 [numbers]
  ;; Operações matemáticas verbosas em map
  (map #(+ 1 %) numbers))  ; Deve sugerir (map inc numbers)

(defn complex-example-4 [values]
  ;; Filtros com verificações verbosas
  (filter #(= false %) values))  ; Deve sugerir (filter false? values)

(defn complex-example-5 [data]
  ;; Verificações verbosas em let
  (let [is-zero (= 0 data)      ; Deve sugerir (zero? data)
        is-nil  (= nil data)    ; Deve sugerir (nil? data)
        incremented (+ data 1)] ; Deve sugerir (inc data)
    [is-zero is-nil incremented]))

;; ============================================================================
;; CASOS EDGE
;; ============================================================================

(defn edge-case-1 [x]
  ;; Comparação com outros números (não deve detectar)
  (= 5 x))

(defn edge-case-2 [x]
  ;; Operação com outros números (não deve detectar)
  (+ 2 x))

(defn edge-case-3 [x]
  ;; Subtração de outros números (não deve detectar)
  (- x 5))

(defn edge-case-4 [x y]
  ;; Comparação entre variáveis (não deve detectar)
  (= x y))

;; ============================================================================
;; EXEMPLOS ANINHADOS
;; ============================================================================

(defn nested-example-1 [coll]
  ;; Verificações verbosas aninhadas
  (if (= 0 (count coll))  ; Deve sugerir (zero? (count coll))
    (= true (empty? coll))  ; Deve sugerir (true? (empty? coll))
    (> (count coll) 0)))    ; Deve sugerir (pos? (count coll))

(defn nested-example-2 [x]
  ;; Operações matemáticas aninhadas
  (+ 1 (- x 1)))  ; Ambas devem ser detectadas: (inc (dec x))

;; ============================================================================
;; EXEMPLOS COM THREADING MACROS
;; ============================================================================

(defn threading-example-1 [x]
  ;; Verificações verbosas em threading
  (-> x
      (+ 1)     ; Deve sugerir inc
      (= 0)))   ; Deve sugerir zero?

(defn threading-example-2 [coll]
  ;; Verificações verbosas em thread-last
  (->> coll
       (filter #(> % 0))  ; Deve sugerir (filter pos? coll)
       (map #(+ 1 %))))   ; Deve sugerir (map inc coll) 