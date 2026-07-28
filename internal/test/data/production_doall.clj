(ns production-doall)

(defn process [items]
  (doall (map inc items)))
