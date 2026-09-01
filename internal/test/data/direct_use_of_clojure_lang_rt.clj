(ns direct-use-of-clojure-lang-rt
  (:import [clojure.lang RT]))

(defn count-items [coll]
  (clojure.lang.RT/count coll))

(defn get-val [m k]
  (RT/get m k))

(defn shadowed-rt [coll]
  (let [RT (fn [& _] nil)]
    (RT/count coll)))

(defn quoted-rt [coll]
  '(clojure.lang.RT/count coll))

(defmacro generated-rt [form]
  `(clojure.lang.RT/count ~form))

(defn syntax-quoted-data [coll]
  `(clojure.lang.RT/count ~coll))
