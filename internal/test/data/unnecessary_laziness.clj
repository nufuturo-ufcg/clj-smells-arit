(ns unnecessary-laziness)

(defn eager-map [xs]
  (vec (map inc xs)))

(defn eager-filter [xs]
  (vec (filter pos? xs)))

(defn qualified-map [xs]
  (vec (clojure.core/map inc xs)))

(defn eager-for [xs]
  (vec (for [x xs] (* x x))))

(defn intentionally-lazy [xs]
  (map inc xs))

(defn eager-mapv [xs]
  (vec (mapv inc xs)))

(defn shadowed-map [map xs]
  (vec (map inc xs)))

(defn unknown-map [xs]
  (vec (custom/map inc xs)))

(def quoted-map
  '(vec (map inc xs)))

;; (vec (map inc xs))

(def discarded-map
  #_(vec (map inc xs)))

(defn into-is-owned-by-another-rule [xs]
  (into [] (map inc xs)))
