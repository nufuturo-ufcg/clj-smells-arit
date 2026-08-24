(ns map-with-nil-values)

(defn destructured [{:keys [value] :or {value nil}}]
  value)

(defn ordinary-result []
  {:optional nil
   :status :ok})

(defn explicit-assoc [m]
  (assoc m :optional nil))
