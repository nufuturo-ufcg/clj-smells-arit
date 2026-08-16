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
