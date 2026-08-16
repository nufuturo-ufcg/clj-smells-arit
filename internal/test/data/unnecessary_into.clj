(ns unnecessary-into-test)

;; Positive: one lazy producer, one source, and a plain vector target.
(defn fuse-map [coll]
  (into [] (map inc coll)))

(defn fuse-filter [coll]
  (into [] (filter even? coll)))

(defn fuse-take [coll]
  (into [] (take 10 coll)))

(defn fuse-distinct [coll]
  (into [] (clojure.core/distinct coll)))

;; Negative: generic conversions do not preserve all observable properties.
(defn unknown-to-vector [coll]
  (into [] coll))

(defn unknown-to-set [coll]
  (into #{} coll))

(defn unknown-to-map [entries]
  (into {} entries))

;; Negative: set/map targets require additional duplicate and hashing proofs.
(defn map-into-set [coll]
  (into #{} (map :id coll)))

(defn map-into-map [m]
  (into {} (map (fn [[k v]] [k (inc v)]) m)))

;; Negative: eager and parallel producers have no transducer-equivalent arity.
(defn eager-map [coll]
  (into [] (mapv inc coll)))

(defn parallel-map [coll]
  (into [] (pmap inc coll)))

;; Negative: map over multiple sources cannot become one map transducer source.
(defn multiple-sources [left right]
  (into [] (map vector left right)))

;; Negative: this already uses into's transducer arity.
(defn already-transduced [coll]
  (into [] (map inc) coll))

;; Negative: unknown destinations can have observable conj behavior.
(defn custom-target [target coll]
  (into target (map inc coll)))

;; Negative: local shadowing must not inherit clojure.core semantics.
(defn shadowed-into [into coll]
  (into [] (map inc coll)))

(defn shadowed-map [map coll]
  (into [] (map inc coll)))

;; Negative: comprehensions and arbitrary lazy producers are unsupported.
(defn comprehension [coll]
  (into [] (for [x coll] (inc x))))

(defn arbitrary-producer [coll]
  (into [] (flatten coll)))

;; Negative: quoted examples are not executed.
(def quoted-example
  '(into [] (map inc coll)))
