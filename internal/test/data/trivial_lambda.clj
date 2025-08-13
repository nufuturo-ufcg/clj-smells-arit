(ns trivial-lambda)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Basic 'map' pattern - single argument forwarder
(defn map-double [xs]
  (map #(inc %) xs))

;; Example 2: Basic 'filter' pattern - predicate forwarding
(defn filter-positives [xs]
  (filter #(pos? %) xs))

;; Example 3: 'reduce' pattern with trivial operator wrapper
(defn sum-list [xs]
  (reduce #(+ %1 %2) 0 xs))

;; Example 4: 'swap!' usage - simple forwarding to a function
(def counter (atom 0))
(defn increment-counter []
  (swap! counter #(inc %)))

;; Example 5: 'update' usage - trivial function wrapper
(defn update-age [person]
  (update person :age #(inc %)))

;; Example 6: 'sort-by' with direct key function
(defn sort-by-length [strings]
  (sort-by #(count %) strings))

;; Example 8: Core.async style mapping (hypothetical)
(defn async-map-upper [ch]
  (map< #(clojure.string/upper-case %) ch))

;; Example 9: Trivial lambdas chained in threading macro (two-findings)
(defn process-numbers [xs]
  (->> xs
       (map #(inc %))
       (filter #(pos? %))))


;; Example 10: Lambda uses subset of arguments
(defn ignore-second [pairs]
  (map #(first %) pairs))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Argument reordering inside lambda
(defn swap-args [pairs]
  (map #(vector (second %) (first %)) pairs))

;; Example 12: Lambda adds constant argument
(defn add-constant [xs]
  (map #(inc (+ % 5)) xs))

;; Example 13: Lambda captures a local variable
(defn multiply-by-factor [xs factor]
  (map #(* % factor) xs))

;; Example 14: Lambda composes multiple functions
(defn compose-functions [xs]
  (map #(-> % inc (* 2)) xs))

;; Example 15: Multiple expressions in lambda
(defn print-and-double [xs]
  (map (fn [x] (println x) (* 2 x)) xs))

;; Example 16: Lambda with condition inside
(defn condition-inside [xs]
  (map #(if (pos? %) (inc %) %) xs))

;; Example 17: Non-trivial lambda in 'reduce'
(defn reduce-with-check [xs]
  (reduce (fn [acc x] (if (pos? x) (+ acc x) acc)) 0 xs))

;; Example 18: 'pmap' pattern - single argument forwarder (not-detected)
(defn pmap-square [xs]
  (pmap #(* % %) xs))







