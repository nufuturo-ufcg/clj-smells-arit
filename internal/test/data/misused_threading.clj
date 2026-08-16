(ns misused-threading-test
  (:require [clojure.string :as str]))

;; Positive: two or more resolved steps consistently expect the opposite side.
(defn wrong-first [xs]
  (-> xs
      (map inc)
      (filter even?)
      (take 5)))

(defn wrong-last [m]
  (->> m
       (assoc :active true)
       (dissoc :temporary)))

(defn wrong-some-first [xs]
  (some-> xs
          (map inc)
          (remove neg?)))

(defn wrong-some-last [m]
  (some->> m
           (update :count inc)
           (select-keys [:count])))

;; Negative: the selected direction agrees with resolved data positions.
(defn correct-first [m]
  (-> m (assoc :a 1) (dissoc :b) (update :c inc)))

(defn correct-last [xs]
  (->> xs (map inc) (filter even?) (take 5)))

;; Negative: anonymous functions can be the clearest exact positioning tool.
(defn necessary-lambdas [context query]
  (->> query
       (#(if (:skip? %) % (assoc % :context context)))
       (#(select-keys % [:context :skip?]))))

;; Negative: heterogeneous result types are not positional evidence.
(defn heterogeneous [m]
  (-> m :name str/upper-case count pos?))

;; Negative: a single contrary function is insufficient evidence.
(defn single-contrary-step [xs]
  (-> xs (map inc)))

;; Negative: mixed resolved directions do not support one safe recommendation.
(defn mixed-directions [x]
  (-> x (assoc :a 1) (map identity) (filter some?)))

;; Negative: unary functions work identically in either macro.
(defn unary-steps [xs]
  (->> xs distinct vec count))

;; Negative: into has two meaningful collection positions (target and source).
(defn accumulating-into [key-set m]
  (-> key-set
      (into (keys m))
      (into (vals m))))

;; Negative: project functions have no assumed core argument semantics.
(defn custom-map [map xs]
  (-> xs (map inc) (map dec)))

;; Negative: a locally shadowed threading symbol is an ordinary function call.
(defn shadowed-thread [-> x]
  (-> x (map inc) (filter even?)))

;; Negative: incomplete forms do not establish a valid expanded arity.
(defn incomplete-steps [x]
  (->> x (assoc) (dissoc)))

;; Negative: quoted examples are not executable pipelines.
(def quoted-example
  '(-> xs (map inc) (filter even?)))
