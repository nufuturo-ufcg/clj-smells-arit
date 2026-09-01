(ns data.single-segment-namespace)

(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))

(def quoted-namespace
  '(ns quoted-namespace))

(defmacro generated-namespace []
  `(ns generated-namespace))

(ns single-segment-namespace)

(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))
