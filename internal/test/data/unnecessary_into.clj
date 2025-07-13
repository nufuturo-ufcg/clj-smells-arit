(ns unnecessary-into)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Converting to vector with into instead of vec
(defn to-vector [xs]
  (into [] xs))

;; Example 2: Building a set with into instead of set
(defn to-into [xs]
  (into #{} xs))

;; Example 3: Redundant map-to-map transformation with into
(defn transform-map-values [m]
  (into {} (map (fn [[k v]] [k (inc v)]) m)))

;; Example 4: Eager realization with into instead of threaded vec
(defn filter-evens [xs]
  (into [] (filter even? xs)))

;; Example 5: Mapping with into instead of vec
(defn increment-all [xs]
  (into [] (map inc xs)))

;; Example 6: Flattening nested collections via apply and into
(defn flatten-lists [xss]
  (into [] (apply concat xss)))

;; Example 7: Reversing collection with into instead of vec
(defn reverse-vector [xs]
  (into [] (reverse xs)))

;; Example 8: Using into for key extraction instead of vec
(defn get-keys [m]
  (into [] (keys m)))

;; Example 9: Using into for mapping and appending instead of transducer
(defn map-append [coll xs]
  (into coll (map str xs)))

;; Example 11: Using into for filtering intead of transducer arity
(defn filter-positive [coll xs]
  (into coll (filter pos? xs)))

;; Example 12: Mapping to vector pairs with into instead of vec
(defn map-to-pairs [m]
  (into [] (map (fn [[k v]] [k v]) m)))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 1: Merging two vectors efficiently
(defn append-vectors [a b]
  (into a b))

;; Example 2: Combining sets
(defn union-sets [a b]
  (into a b)) ;; same as clojure.set/union

;; Example 3: Transducing with filtering and mapping
(defn even-squares [xs]
  (into [] (comp (filter even?) (map #(* % %))) xs))

;; Example 4: Adding elements to a non-empty map
(defn extend-map [m kvs]
  (into m kvs))

;; Example 5: Using into with a transducer and initial set
(defn odd-set [xs]
  (into #{} (filter odd?) xs))

;; Example 6: Preparing a prefix sequence
(defn prepend-items [prefix coll]
  (into (vec prefix) coll))

;; Example 7: Transducing with multiple sets
(defn transform [xs]
  (into [] (comp (map inc) (filter odd?)) xs))

;; Example 8: Building a list with into from vector
(defn vector-to-list [v]
  (into '() v))