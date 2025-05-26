(ns test.file3)

(defn compute-results [numbers]
  (let [filtered (filter pos? numbers)
        doubled (map #(* 2 %) filtered)
        total (reduce + doubled)]
    {:filtered filtered
     :doubled doubled
     :total total}))

(defn verify-data [dataset]
  (when (and dataset (not-empty dataset))
    (every? number? dataset)))

(defn another-unique [a b c]
  (if (> a b)
    (+ a c)
    (- b c))) 