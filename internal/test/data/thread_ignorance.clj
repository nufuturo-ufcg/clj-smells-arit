(ns thread-ignorance)

(defn process-data [x]
  (map inc (filter pos? (take 10 x))))
