(ns test.clean2)

(defn handle-numbers [values]
  (let [filtered (filter pos? values)
        doubled (map #(* 2 %) filtered)
        total (reduce + doubled)]
    {:filtered filtered
     :doubled doubled
     :total total}))

(defn check-data [input]
  (when (and input (not-empty input))
    (every? number? input))) 