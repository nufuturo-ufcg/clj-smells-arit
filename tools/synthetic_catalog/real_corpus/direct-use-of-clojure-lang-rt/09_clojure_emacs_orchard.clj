(ns orchard.inspect.analytics
  "Submodule of Orchard Inspector for getting quick insights about the inspected
  data. A \"Metabase\" for Orchard/CIDER Inspector."
  (:refer-clojure :exclude [bounded-count])
  (:require
   [clojure.string :as str])
  (:import
   (clojure.lang RT)
   (java.util List Map Set)))

(definline ^:private inc-if [val condition]
  `(cond-> ~val ~condition inc))

(defn- count-pred [pred limit ^Iterable coll]
  (let [it (RT/iter coll)]
    (loop [i 0, n 0]
      (if (and (< i limit) (.hasNext it))
        (let [x (.next it)]
          (recur (inc i) (inc-if n (pred x))))
        [n (if (pos? i) (/ n i) 0.0)]))))
