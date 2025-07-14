(ns inefficient-filtering)

;; Sample datasets
(def users
  [{:id 1 :name "Alice" :age 30}
   {:id 2 :name "Bob" :age 40}
   {:id 3 :name "Alice" :age 25}
   {:id 4 :name "Eve" :age 20}])

(def numbers [1 2 3 4 5 6 7 8 9 10])
(def log-entries
  [{:level :info :message "Started"}
   {:level :warn :message "Something odd"}
   {:level :error :message "Crashed"}])

;; ===================================================
;; GROUP 1: COMMON INEFFICIENT FILTERING SMELLS
;; ===================================================

(defn find-first-by-filter []
  ;; Smell: filter + first
  (first (filter #(= (:name %) "Alice") users)))

(defn count-filtered-values []
  ;; Smell: builds intermediate collection only to count
  (count (filter even? numbers)))

(defn empty-after-filter []
  ;; Smell: full filter, then checks if empty
  (empty? (filter #(> (:age %) 18) users)))

(defn redundant-double-filter []
  ;; Smell: two filters instead of one
  (->> users
       (filter #(> (:age %) 18))
       (filter #(= (:name %) "Alice"))))

(defn check-if-any-underage? []
  ;; Smell: uses not-empty after filter
  (not-empty (filter #(< (:age %) 18) users)))

(defn count-positive-redundant []
  ;; Smell: same as earlier, counts using filter
  (count (filter pos? numbers)))


;; ===================================================
;; GROUP 2: INTERMEDIATE-LEVEL FILTERING SMELLS
;; ===================================================

(defn map-after-filter []
  ;; Smell: filtering then mapping — could be `keep`
  (map :name (filter #(> (:age %) 20) users)))

(defn multiple-pass-using-remove []
  ;; Smell: filter + remove = two full traversals
  (->> users
       (filter #(> (:age %) 20))
       (remove #(= (:name %) "Bob"))))

(defn filter-then-take []
  ;; Smell: filters everything, then takes 2
  (take 2 (filter odd? numbers)))

(defn select-first-names-over-25 []
  ;; Smell: maps after filter, could short-circuit with transducer
  (map :name (filter #(> (:age %) 25) users)))

(defn extract-warnings []
  ;; Smell: filters all logs even if we need only one
  (first (filter #(= (:level %) :warn) log-entries)))

;; ===================================================
;; GROUP 3: REFACTORED (IMPROVED) VERSIONS
;; ===================================================

(defn find-first-by-filter-refactored []
  (some #(when (= (:name %) "Alice") %) users))

(defn count-filtered-values-refactored []
  (reduce (fn [acc x] (if (even? x) (inc acc) acc)) 0 numbers))

(defn empty-after-filter-refactored []
  (not (some #(> (:age %) 18) users)))

(defn redundant-double-filter-refactored []
  (filter #(and (> (:age %) 18)
                (= (:name %) "Alice")) users))

(defn check-if-any-underage?-refactored []
  (some #(< (:age %) 18) users))

(defn count-positive-redundant-refactored []
  (reduce (fn [acc x] (if (pos? x) (inc acc) acc)) 0 numbers))


;; ---------------------------------------------------

(defn map-after-filter-refactored []
  ;; `keep` does map + filter
  (keep #(when (> (:age %) 20) (:name %)) users))

(defn multiple-pass-using-remove-refactored []
  ;; Single filter with composed predicate
  (filter #(and (> (:age %) 20)
                (not= (:name %) "Bob")) users))

(defn filter-then-take-refactored []
  ;; Transducer short-circuits when using into or sequence
  (sequence (comp (filter odd?) (take 2)) numbers))

(defn select-first-names-over-25-refactored []
  ;; Single pass using keep
  (keep #(when (> (:age %) 25) (:name %)) users))

(defn extract-warnings-refactored []
  ;; Stops early with `some`
  (some #(when (= (:level %) :warn) %) log-entries))
