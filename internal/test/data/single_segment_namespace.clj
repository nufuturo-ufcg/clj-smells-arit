(ns data.single-segment-namespace)

(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))

(ns single-segment-namespace)

(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))
