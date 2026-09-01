(ns immutability-violation-alter-var-root)

(def config {})

(defn update-config [value]
  (alter-var-root #'config assoc :value value))

(defn invalid-arity []
  (alter-var-root))

(defn unresolved-operation [value]
  (custom/alter-var-root #'config identity value))

(defn shadowed-operation [alter-var-root value]
  (alter-var-root #'config identity value))

(def quoted-operation
  '(alter-var-root #'config identity))

(comment
  (alter-var-root #'config identity))
