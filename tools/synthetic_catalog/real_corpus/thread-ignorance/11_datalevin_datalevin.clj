(defn keyword->string
  [k]
  (if-let [n (namespace k)]
    (str n "/" (name k))
    (name k)))

(defn attr-domain
  [attr]
  (str/replace (keyword->string attr) "/" "_"))

(defn vector-attr-domains
  [attr props]
  (when (identical? :db.type/vec (:db/valueType props))
    (vec (distinct (conj (vec (:db.vec/domains props)) (attr-domain attr))))))
