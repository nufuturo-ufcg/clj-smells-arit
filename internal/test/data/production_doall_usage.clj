(ns production-doall-usage)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Eagerly Printing Items with doall
(defn print-items [items]
  (doall (map println items)))

;; Example 2: Redundant doall in Mapping and Filtering
(defn print-evens []
  (doall (map #(println %) (filter even? (range 1000)))))

;; Example 3: Forcing Realization of Nested Maps
(defn extract-values [m]
  (doall
   (mapcat vals (vals m))))

;; Example 4: Combining doall and doseq Unnecessarily
(defn log-users [users]
  (doall
   (doseq [u users]
     (println "User:" u))))

;; Example 5: Premature Realization of Remote URLs
(defn fetch-data [ids]
  (doall
   (map #(str "https://api.site.com/item/" %) ids)))

;; Example 6: Realizing a Large Range in Memory
(defn build-list []
  (let [xs (doall (map inc (range 1e6)))]
    (reduce + xs)))

;; Example 7: Recursively Forcing Realization in Loops
(defn deep-process [n]
  (if (zero? n)
    []
    (let [results (doall (map dec (range n)))]
      (conj (deep-process (dec n)) results))))

;; Example 8: Eager Evaluation of Indexed Mapping
(defn tag-lines [lines]
  (doall
   (map-indexed #(str %1 ": " %2) lines)))

;; Example 9: Conditional Logging with Forced Evaluation
(defn maybe-log [enabled? msgs]
  (when enabled?
    (doall (map println msgs))))

;; Example 10: Premature Realization of Partitioned Sequences
(defn process-in-pairs [coll]
  (doall
   (map (fn [[a b]] (+ a b))
        (partition 2 coll))))
