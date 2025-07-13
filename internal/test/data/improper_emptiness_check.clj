(ns improper-emptiness-check)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Negated empty?
(defn has-elements? [xs]
  (not (empty? xs)))

;; Example 2: Comparing count to zero
(defn is-empty? [xs]
  (= 0 (count xs)))

;; Example 3: Checking non-emptiness with count
(defn has-any? [xs]
  (> (count xs) 0))

;; Example 4: Checking emptiness with not and seq
(defn is-empty-seq [xs]
  (not (seq xs)))

