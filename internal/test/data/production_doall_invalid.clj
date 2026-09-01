(ns production-doall-invalid)

;; Invalid arities are not executable doall occurrences.
(defn missing-doall-argument []
  (doall))

(defn missing-mapv-collection [items]
  (doall (mapv inc)))

(defn extra-vec-argument [items]
  (doall (vec items :unexpected)))

(defn missing-into-source [items]
  (doall (into [])))
