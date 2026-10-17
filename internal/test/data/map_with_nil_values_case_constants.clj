(ns map-with-nil-values-case-constants)

;; case test constants are unevaluated literal data; they build no map.
(defn constant [x]
  (case x
    {:a nil} :matched
    :other))

;; Result and default branches are evaluated map literals.
(defn result-branch [x]
  (case x
    :k {:a nil}
    {:b nil}))

;; Unquoted code in a case template runs when the template is built; it is not
;; a constant.
(defn assoc-case-template [k fields]
  `(case ~k
     ~@(mapcat (fn [f] [f (replace {f nil} fields)]) fields)
     nil))
