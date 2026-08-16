(ns test-macro)

(defn process-thread-last [data f]
  (->> data
       (map f)
       (into [])))

(defn process-thread-first [data]
  (-> data
      (into [])
      (into {})))
