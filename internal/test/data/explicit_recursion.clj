(ns explicit-recursion)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Basic 'map' pattern - simple transformation
(defn double-nums-recursive [nums]
  (if (empty? nums)
    '()
    (cons (* 2 (first nums)) (double-nums-recursive (rest nums)))))

;; Example 2: Basic 'reduce' pattern - accumulation
(defn sum-list [numbers]
  (if (empty? numbers)
    0
    (+ (first numbers) (sum-list (rest numbers)))))

;; Example 3: 'filter' pattern with let binding
(defn get-even-numbers [coll]
  (if (empty? coll)
    []
    (let [f (first coll)
          r (rest coll)]
      (if (even? f)
        (cons f (get-even-numbers r))
        (get-even-numbers r)))))

;; Example 4: 'map' pattern with more complex function
(defn square-and-add-one [nums]
  (if (empty? nums)
    []
    (cons (+ 1 (* (first nums) (first nums)))
          (square-and-add-one (rest nums)))))

;; Example 5: 'reduce' pattern with multiplication
(defn product-list [numbers]
  (if (empty? numbers)
    1
    (* (first numbers) (product-list (rest numbers)))))

;; Example 6: 'filter' pattern direct (without let)
(defn get-positive [coll]
  (if (empty? coll)
    []
    (if (pos? (first coll))
      (cons (first coll) (get-positive (rest coll)))
      (get-positive (rest coll)))))

;; Example 7: 'map' pattern with strings
(defn uppercase-strings [strings]
  (if (empty? strings)
    []
    (cons (clojure.string/upper-case (first strings))
          (uppercase-strings (rest strings)))))

;; Example 8: 'reduce' pattern with string concatenation
(defn concat-all [strings]
  (if (empty? strings)
    ""
    (str (first strings) (concat-all (rest strings)))))

;; Example 9: 'filter' pattern with custom predicate
(defn get-long-words [words]
  (if (empty? words)
    []
    (let [word (first words)
          rest-words (rest words)]
      (if (> (count word) 5)
        (cons word (get-long-words rest-words))
        (get-long-words rest-words)))))

;; Example 10: 'map' pattern with type conversion
(defn strings-to-ints [str-nums]
  (if (empty? str-nums)
    []
    (cons (Integer/parseInt (first str-nums))
          (strings-to-ints (rest str-nums)))))

;; Example 11: 'reduce' pattern with boolean operation
(defn all-true? [bools]
  (if (empty? bools)
    true
    (and (first bools) (all-true? (rest bools)))))

;; Example 12: 'filter' pattern with multiple conditions
(defn get-valid-numbers [nums]
  (if (empty? nums)
    []
    (let [n (first nums)]
      (if (and (number? n) (pos? n) (< n 100))
        (cons n (get-valid-numbers (rest nums)))
        (get-valid-numbers (rest nums))))))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 13: Legitimate recursion - tree processing
(defn tree-depth [tree]
  (if (coll? tree)
    (inc (apply max (map tree-depth tree)))
    0))

;; Example 14: Legitimate recursion - nested structure navigation
(defn find-in-nested [nested-map key]
  (cond
    (map? nested-map) (or (get nested-map key)
                          (some #(find-in-nested % key) (vals nested-map)))
    (coll? nested-map) (some #(find-in-nested % key) nested-map)
    :else nil))

;; Example 15: Legitimate recursion - specific algorithm (Fibonacci)
(defn fibonacci [n]
  (if (<= n 1)
    n
    (+ (fibonacci (- n 1)) (fibonacci (- n 2)))))

;; Example 16: Legitimate recursion - processing with multiple parameters
(defn merge-sorted [list1 list2]
  (cond
    (empty? list1) list2
    (empty? list2) list1
    (<= (first list1) (first list2))
    (cons (first list1) (merge-sorted (rest list1) list2))
    :else
    (cons (first list2) (merge-sorted list1 (rest list2)))))

;; Example 17: Legitimate recursion - quicksort (uses cond instead of if to avoid detection)
(defn quicksort [coll]
  (cond
    (empty? coll) []
    :else
    (let [pivot (first coll)
          rest-coll (rest coll)
          smaller (filter #(< % pivot) rest-coll)
          greater (filter #(>= % pivot) rest-coll)]
      (concat (quicksort smaller) [pivot] (quicksort greater)))))

;; Example 18: Non-recursive function (should not be detected)
(defn simple-double [x]
  (* 2 x))

;; Example 19: Mutual recursion (should not be detected by this rule)
(declare odd-helper)
(defn even-helper [n]
  (if (zero? n)
    true
    (odd-helper (dec n))))

(defn odd-helper [n]
  (if (zero? n)
    false
    (even-helper (dec n))))

;; Example 20: Legitimate recursion - infinite sequence generation
(defn generate-sequence [start step]
  (lazy-seq
    (cons start (generate-sequence (+ start step) step)))) 
