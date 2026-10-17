(ns verbose-checks-boolean-identity)

;; On the JVM a non-canonical java.lang.Boolean is = to the literal but not
;; identical to it, so these rewrites depend on the operand's contract.
(defn unknown-operand [flag opts]
  [(= true flag)
   (= (:enabled opts) false)
   (if flag true false)])

;; Core predicates always return the canonical Boolean.
(defn canonical-operand [xs n]
  [(= true (empty? xs))
   (= false (pos? n))
   (if (some? n) true false)])

;; (if x false true) is the definition of not.
(defn negation [flag]
  (if flag false true))

;; A local predicate is not clojure.core's.
(defn shadowed-predicate [empty? xs]
  (= true (empty? xs)))
