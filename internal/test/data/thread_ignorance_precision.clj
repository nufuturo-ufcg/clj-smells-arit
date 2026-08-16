(ns thread-ignorance-precision
  (:require [clojure.string :as string]))

;; Mixed first/last positions cannot use one threading macro.
(defn mixed-direction [m]
  (map inc (keys (assoc m :active true))))

;; The nested value is not in filter's collection position.
(defn wrong-position [xs]
  (filter (first xs) (rest xs)))

;; Two list arguments make the data path ambiguous.
(defn branching-call [xs]
  (map (comp inc first) (filter pos? xs)))

;; Project functions have no known argument contract.
(defn project-functions [x]
  (custom-d (custom-c (custom-b (custom-a x)))))

;; Descriptive domain names are useful documentation, not disposable temps.
(defn domain-let [headers]
  (let [search-after (get headers "search-after")
        decoded-search-after (decode search-after)
        request-context (assoc {} :search-after decoded-search-after)]
    request-context))

;; A reused intermediate would be duplicated or lost by a rewrite.
(defn reused-let [xs]
  (let [step1 (filter pos? xs)
        step2 (map inc step1)
        step3 (concat step1 step2)]
    step3))

;; The last result is not returned directly.
(defn transformed-body [xs]
  (let [step1 (filter pos? xs)
        step2 (map inc step1)
        step3 (vec step2)]
    {:items step3}))

;; Destructuring and type hints intentionally remain outside the contract.
(defn destructured-let [xs]
  (let [step1 (filter pos? xs)
        [step2] (map inc step1)
        step3 (vec step2)]
    step3))

(defn hinted-let [xs]
  (let [step1 (filter pos? xs)
        ^java.util.List step2 (mapv inc step1)
        step3 (vec step2)]
    step3))

;; Existing threading must not be reported.
(defn already-threaded [xs]
  (->> xs
       (take 10)
       (filter pos?)
       (map inc)
       vec))

;; Deferred, quoted and anonymous-function bodies are excluded.
(defn non-immediate [xs]
  (delay (vec (map inc (filter pos? (take 10 xs)))))
  '(vec (map inc (filter pos? (take 10 xs))))
  (fn [] (vec (map inc (filter pos? (take 10 xs))))))

;; A shorter chain remains below the configured threshold.
(defn short-chain [xs]
  (map inc (filter pos? xs)))

;; A local binding named like clojure.core/map must not inherit core semantics.
(defn shadowed-core-name [xs]
  (let [map custom-map]
    (vec (map inc (filter pos? (take 10 xs))))))
