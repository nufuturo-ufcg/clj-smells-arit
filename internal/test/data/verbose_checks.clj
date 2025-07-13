(ns verbose-checks)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Checking for zero with =
(defn is-zero [n]
  (= n 0))

;; Example 2: Checking for positivity verbosely
(defn is-positive [n]
  (> n 0))

;; Example 3: Checking for negativity verbosely
(defn is-negative [n]
  (< n 0))

;; Example 4: Checking boolean true with =
(defn is-true [x]
  (= true x))

;; Example 5: Checking boolean false with =
(defn is-false [x]
  (= false x))

;; Example 6: Verbose nil check with =
(defn is-nil [x]
  (= nil x))

;; Example 7: Inverted positivity check
(defn not-positive? [n]
  (not (> n 0)))

;; Example 8: Redundant boolean check in condition
(defn should-run? [flag]
  (if (= true flag) :yes :no))

;; Example 9: Length comparision with explicit zero
(defn empty-coll? [coll]
  (= (count coll) 0))

;; Example 10: Reversed zero comparison
(defn is-zero-alt [n]
  (= 0 n))

;; Example 11: Reversed positive comparison
(defn is-positive-alt [n]
  (< 0 n))

;; Example 12: Negated nil check with not and =
(defn present? [x]
  (not (= x nil)))

;; Example 13: Explicit comparison inside cond
(defn signal [v]
  (cond
    (= v true)  :go
    (= v false) :stop
    :else       :unknown))

;; Example 14: Multiple verbose checks combined
(defn validate [n x]
  (and
   (not (= n 0))
   (not (= x nil))
   (not (= x false))))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 15: Checking greather than one
(defn multiple-items? [xs]
  (> (count xs) 1))

;; Example 16: Between two values
(defn in-range? [n]
  (and (<= 0 n) (<= n 10)))

;; Example 17: Comparing two arbitrary numbers
(defn greater-than? [a b]
  (> a b))

;; Example 18: Checking for specific value (not zero)
(defn is-one? [n]
  (= n 1))