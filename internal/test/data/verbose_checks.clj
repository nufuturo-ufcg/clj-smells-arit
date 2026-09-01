(ns verbose-checks)

(defn checks [x]
  (if (= 0 x)
    (println "zero"))
  (if (= x true)
    (println "true"))
  (if (= nil x)
    (println "nil"))
  (+ 1 x))

;; These rewrites are not universally semantics-preserving.
(defn preserve-integer-contract [x]
  (>= x 0))

(defn preserve-truthiness-contract [x]
  (not= x false))

;; Core names may be shadowed by parameters.
(defn shadowed-comparison [= x]
  (= 0 x))

;; An explicit integral coercion makes = 0 -> zero? safe.
(defn integral-zero [x]
  (= 0 (long x)))

;; Equality with nil can be replaced by the boolean some? predicate.
(defn non-nil [x]
  (not= nil x))

;; String literals prove the integer return type of these methods.
(defn literal-string-index []
  (= (.indexOf "text with needle" "needle") 0))

(defn literal-string-length []
  (> (.length "text") 0))

(defn literal-vector-size []
  (> (.size [1 2 3]) 0))

(defn literal-vector-index []
  (= (.indexOf [1 2 3] 2) 0))

(defn literal-map-size []
  (> (.size {:a 1}) 0))

(defn literal-set-size []
  (> (.size #{:a}) 0))

(defn dynamic-collection-size [coll]
  (> (.size coll) 0))

;; Invalid producer arities do not establish an executable integral expression.
(defn invalid-count-arity [x]
  (= 0 (count)))

(defn invalid-length-arity []
  (= 0 (.length "text" :extra)))

;; The suggested core replacement may itself be shadowed lexically.
(defn shadowed-zero-replacement [zero? x]
  (= 0 x))

(defn shadowed-inc-replacement [inc x]
  (+ 1 x))

(defn shadowed-some-replacement [some? x]
  (not= nil x))

(defn shadowed-boolean-replacement [boolean x]
  (if x true false))

(defn shadowed-zero-in-parity-replacement [zero? even? x]
  (= (mod x 2) 0))

;; Constant non-numeric operands must not be rewritten to numeric predicates.
(defn literal-nil-numeric []
  (= 0 nil))

(defn literal-string-numeric []
  (= 0 "zero"))

;; These forms remain contextual because their numeric domain is not proven.
(defn nan-numeric [x]
  (= 0 Double/NaN))

(defn overflow-numeric [x]
  (= 0 (+ x 9223372036854775807)))

(defn wrong-type-hint [^String x]
  (= 0 x))

;; Invalid integral-call arities must not establish a numeric operand.
(defn invalid-compare-arity [x]
  (= 0 (compare x)))
