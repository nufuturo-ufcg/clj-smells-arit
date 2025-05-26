(ns test.simple-dup)

(defn calculate-total [items]
  (let [sum (reduce + 0 items)
        tax (* sum 0.1)
        total (+ sum tax)]
    {:sum sum
     :tax tax
     :total total}))

(defn compute-amount [values]
  (let [sum (reduce + 0 values)
        tax (* sum 0.1)
        total (+ sum tax)]
    {:sum sum
     :tax tax
     :total total})) 