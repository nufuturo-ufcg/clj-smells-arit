(ns inappropriate-collection
  "Examples of inappropriate collection usage in Clojure")

;; ========================================
;; 1. BASIC PATTERNS
;; ========================================

;; Using nth on list (O(n) random access)
(defn bad-random-access []
  (nth '(1 2 3 4 5) 3))  ; O(n) - should use vector

;; Using some for membership check on vector
(defn bad-membership-check []
  (some #(= % "admin") ["user" "admin" "guest"]))  ; O(n) - should use set

;; Using contains? on vector with non-numeric key
(defn bad-vector-contains []
  (contains? ["a" "b" "c"] "b"))  ; always false - should use set

;; ========================================
;; 2. FILTERING PATTERNS
;; ========================================

;; first + filter (inefficient)
(defn bad-first-filter [users]
  (first (filter #(= (:role %) "admin") users)))  ; processes entire collection
;; Better: (some #(when (= (:role %) "admin") %) users)

;; empty? + filter (inefficient)
(defn bad-empty-filter [users]
  (empty? (filter #(= (:role %) "admin") users)))  ; processes entire collection
;; Better: (not-any? #(= (:role %) "admin") users)

;; count + filter (inefficient for counting)
(defn bad-count-filter [users]
  (count (filter #(= (:status %) "active") users)))  ; processes entire collection
;; Better: (transduce (filter #(= (:status %) "active")) (completing (fn [acc _] (inc acc))) 0 users)

;; ========================================
;; 3. CONCATENATION PATTERNS
;; ========================================

;; sequence + mapcat (memory issues)
(defn bad-sequence-mapcat [nested-data]
  (sequence (mapcat identity) nested-data))  ; can consume too much memory
;; Better: (mapcat identity nested-data) or (transduce cat conj nested-data)

;; nested concat (inefficient)
(defn bad-nested-concat [a b c d]
  (concat a (concat b (concat c d))))  ; deep call stack
;; Better: (into a (concat b c d)) or (reduce into [a b c d])

;; apply concat (inefficient)
(defn bad-apply-concat [nested-colls]
  (apply concat nested-colls))  ; use mapcat
;; Better: (mapcat identity nested-colls)

;; ========================================
;; 4. TRANSFORMATION PATTERNS
;; ========================================

;; map with identity (unnecessary)
(defn bad-map-identity [coll]
  (map identity coll))  ; unnecessary
;; Better: (seq coll) or simply coll

;; reverse on lazy sequence (forces realization)
(defn bad-reverse-lazy [coll]
  (reverse (map inc coll)))  ; forces complete realization
;; Better: use into with preserved order or rethink algorithm

;; for trivial when map would be better
(defn bad-for-trivial [coll]
  (for [x coll] (str x)))  ; use map
;; Better: (map str coll)

;; ========================================
;; 5. NEGATION PATTERNS
;; ========================================

;; filter with not (use remove)
(defn bad-filter-not [coll]
  (filter (comp not odd?) coll))  ; less clear
;; Better: (remove odd? coll)

;; remove with not (double negation)
(defn bad-remove-not [coll]
  (remove (comp not even?) coll))  ; confusing
;; Better: (filter even? coll)

;; ========================================
;; 6. CONVERSION PATTERNS
;; ========================================

;; into [] when vec would be clearer
(defn bad-into-vector [coll]
  (into [] coll))  ; less clear
;; Better: (vec coll)

;; into #{} when set would be clearer
(defn bad-into-set [coll]
  (into #{} coll))  ; less clear
;; Better: (set coll)

;; ========================================
;; 7. REALIZATION PATTERNS
;; ========================================

;; doall with map (dangerous in production)
(defn bad-doall-map [coll]
  (doall (map println coll)))  ; dangerous
;; Better: (mapv println coll) or (transduce ...)

;; ========================================
;; 8. EMPTINESS CHECK PATTERNS
;; ========================================

;; (= 0 (count coll)) when empty? would be better
(defn bad-count-zero [coll]
  (= 0 (count coll)))  ; less idiomatic
;; Better: (empty? coll)

;; (> (count coll) 0) when seq would be better
(defn bad-count-positive [coll]
  (> (count coll) 0))  ; less idiomatic
;; Better: (seq coll)

;; (not (empty? coll)) when seq would be better
(defn bad-not-empty [coll]
  (not (empty? coll)))  ; less idiomatic
;; Better: (seq coll)

;; ========================================
;; 9. MERGE PATTERNS
;; ========================================

;; merge with many arguments (inefficient)
(defn bad-merge-many [m1 m2 m3 m4 m5]
  (merge m1 m2 m3 m4 m5))  ; can be slow
;; Better: use reduce-kv for specific cases

;; ========================================
;; 10. HIERARCHICAL ACCESS PATTERNS
;; ========================================

;; assoc-in unnecessary for single level
(defn bad-assoc-in-single [m k v]
  (assoc-in m [k] v))  ; unnecessary overhead
;; Better: (assoc m k v)

;; get-in unnecessary for single level
(defn bad-get-in-single [m k]
  (get-in m [k]))  ; unnecessary overhead
;; Better: (get m k)

;; ========================================
;; 11. INDEXING PATTERNS
;; ========================================

;; zipmap with range (inefficient)
(defn bad-zipmap-range [coll]
  (zipmap (range) coll))  ; creates unnecessary range
;; Better: (map-indexed vector coll) depending on usage

;; ========================================
;; 12. GENERATION PATTERNS
;; ========================================

;; take + repeatedly when range would be clearer
(defn bad-take-repeatedly []
  (take 10 (repeatedly #(rand-int 100))))  ; less clear
;; Better: (map (fn [_] (rand-int 100)) (range 10)) if lazy not needed

;; ========================================
;; 13. EXTRACTION PATTERNS
;; ========================================

;; keys + group-by when distinct would be better
(defn bad-keys-group-by [coll]
  (keys (group-by :type coll)))  ; inefficient
;; Better: (distinct (map :type coll))

;; ========================================
;; 14. REDUNDANT PATTERNS
;; ========================================

;; seq + count when empty? would be better
(defn bad-seq-count [coll]
  (not (zero? (count coll))))  ; redundant
;; Better: (empty? coll)

;; ========================================
;; 15. FUTURE PATTERNS (for future implementation)
;; ========================================

;; Heisenparameter - function that accepts item or collection
(defn bad-heisenparameter [input]
  (let [items (if (coll? input) input [input])]  ; ambiguity
    (map str items)))
;; Better: have separate functions or always require collection

;; ========================================
;; 16. CORRECTED VERSIONS
;; ========================================

;; Efficient versions of problematic patterns
(defn good-random-access []
  (nth [1 2 3 4 5] 3))  ; O(1) with vector

(defn good-membership-check []
  (contains? #{"user" "admin" "guest"} "admin"))  ; O(1) with set

(defn good-vector-contains []
  (contains? #{"a" "b" "c"} "b"))  ; correct with set

(defn good-find-first [users]
  (some #(when (= (:role %) "admin") %) users))  ; for early termination

(defn good-any-check [users]
  (not-any? #(= (:role %) "admin") users))  ; early termination

(defn good-count-efficient [users]
  (transduce (filter #(= (:status %) "active"))
             (completing (fn [acc _] (inc acc)))
             0
             users))  ; more efficient counting

(defn good-flatten-data [nested-data]
  (mapcat identity nested-data))  ; direct, clearer

(defn good-combine-colls [a b c d]
  (into a (concat b c d)))  ; more efficient

(defn good-map-identity [coll]
  (seq coll))  ; or simply coll

(defn good-apply-concat [nested-colls]
  (mapcat identity nested-colls))  ; more efficient

(defn good-reverse-lazy [coll]
  (into [] (comp (map inc)) (reverse coll)))  ; using transduction

(defn good-merge-many [maps]
  (reduce merge maps))  ; or reduce-kv for specific cases

(defn good-assoc-in-single [m k v]
  (assoc m k v))  ; direct

(defn good-get-in-single [m k]
  (get m k))  ; direct

(defn good-zipmap-range [coll]
  (map-indexed vector coll))  ; more efficient

(defn good-take-repeatedly []
  (map (fn [_] (rand-int 100)) (range 10)))  ; clearer

(defn good-for-trivial [coll]
  (map str coll))  ; more direct

(defn good-filter-not [coll]
  (remove odd? coll))  ; clearer

(defn good-remove-not [coll]
  (filter even? coll))  ; no double negation

(defn good-into-vector [coll]
  (vec coll))  ; clearer

(defn good-into-set [coll]
  (set coll))  ; clearer

(defn good-doall-map [coll]
  (mapv println coll))  ; forces realization safely

(defn good-count-zero [coll]
  (empty? coll))  ; idiomatic

(defn good-count-positive [coll]
  (seq coll))  ; idiomatic

(defn good-not-empty [coll]
  (seq coll))  ; idiomatic

(defn good-keys-group-by [coll]
  (distinct (map :type coll)))  ; more efficient

(defn good-seq-count [coll]
  (seq coll))  ; correct