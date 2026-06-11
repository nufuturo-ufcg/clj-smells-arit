(ns unnecessary-into)

(defn transform-vec [coll]
  (into [] coll))

(defn transform-set [coll]
  (into #{} coll))

(defn map-vals [m]
  (into {} (map (fn [[k v]] [k (inc v)]) m)))
