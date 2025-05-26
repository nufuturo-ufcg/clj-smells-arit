(ns test.clean1)

(defn process-data [items]
  (let [filtered (filter pos? items)
        doubled (map #(* 2 %) filtered)
        total (reduce + doubled)]
    {:filtered filtered
     :doubled doubled
     :total total}))

(defn validate-input [data]
  (when (and data (not-empty data))
    (every? number? data))) 