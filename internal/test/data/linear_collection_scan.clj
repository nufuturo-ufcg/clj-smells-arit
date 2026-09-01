#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns lcs)

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;; LINEAR COLLECTION SCAN - CLASSIC EXAMPLES
;; Classic examples of collection scanning in Clojure
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;

;;--------------------------------------------------
;; Example 1: Find an element using a manual loop (bad)
(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))

;; Refactored: Using some (idiomatic)
(defn scan-find-good [pred coll]
  (some #(when (pred %) %) coll))

;;--------------------------------------------------
;; Example 2: Count elements that satisfy a predicate (bad)
(defn scan-count-bad [coll pred]
  (count (filter pred coll)))

;; Refactored: Using transduce (efficient)
(defn scan-count-good [coll pred]
  (transduce (filter pred) (completing (fn [acc _] (inc acc))) 0 coll))

;;--------------------------------------------------
;; Example 3: Find the minimum using sort (bad)
(defn scan-min-bad [coll]
  (first (sort coll)))

;; Refactored: Using apply/min (efficient)
(defn scan-min-good [coll]
  (apply min coll))

;;--------------------------------------------------
;; Example 4: Check for existence using filter+count (bad)
(defn scan-exists-bad [x coll]
  (> (count (filter #(= % x) coll)) 0))

;; Refactored: Using some (efficient)
(defn scan-exists-good [x coll]
  (some #(= % x) coll))

;;--------------------------------------------------
;; Example 5: Multiple chained maps (bad)
(defn scan-multi-map-bad [coll]
  (map inc (map #(* % 2) (map abs coll))))

;; Refactored: Function composition (efficient)
(defn scan-multi-map-good [coll]
  (map (comp inc #(* % 2) abs) coll))

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;; END OF FILE
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
