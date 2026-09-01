(ns map-with-nil-values)

(defn destructured [{:keys [value] :or {value nil}}]
  value)

(defn ordinary-result []
  {:optional nil
   :status :ok})

(defn explicit-assoc [m]
  (assoc m :optional nil))

(defn conditional-value [flag]
  {:optional (if flag :present nil)})

(defn shadowed-assoc [assoc m]
  (assoc m :optional nil))

(defn invalid-assoc []
  (assoc :optional nil))

(defmacro generated-map []
  `{:optional nil})
