(ns verbose-checks-method-bodies)
;; Method implementations and arity bodies are evaluated code; an unknown
;; macro may read its arguments as data.
(deftype V [^long total]
  clojure.lang.Seqable
  (seq [this] (when (< 0 total) [total])))
(def inc-fn
  (reify java.util.function.BiFunction
    (apply [this k v] (if-not v 1 (+ 1 (long v))))))
(defn multi
  ([x] (when (> x 0) x))
  ([x y] (when (> x 0) y)))
(defn unknown-macro [x]
  (my/dsl (> x 0)))
