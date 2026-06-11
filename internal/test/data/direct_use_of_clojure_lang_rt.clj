(ns direct-use-of-clojure-lang-rt)

(defn count-items [coll]
  (clojure.lang.RT/count coll))

(defn get-val [m k]
  (RT/get m k))
