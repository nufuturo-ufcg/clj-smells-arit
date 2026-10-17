(ns devtools.formatters.state)

(def ^:dynamic *current-state* nil)

(defn valid-current-state? []
  (some? *current-state*))

(defn get-current-state []
  {:pre [(valid-current-state?)]}
  *current-state*)

(defn get-current-history []
  (:history (get-current-state)))

(defn get-last-object-from-current-history []
  (first (get-current-history)))

(defn present-path-segment [v & [starting-index]]
  (cond
    (string? v) v
    ;; we'd like to preserve keywords for easy get
    (keyword? v) v
    (and (number? v)
         (number? starting-index)) (+ v starting-index)     ;; ensures indexing in very large objects
    (number? v) v
    :else "?"))

(defn seek-path-segment [coll val & [seq'd-map?]]
  (let [starting-index (-> (get-last-object-from-current-history)
                           (meta)
                           :index)
        *              (fn [[k v]]
                         (cond
                           ;; we need to know the paths for keywords, these are clickable
                           (and seq'd-map? (identical? k val))
                           (present-path-segment k)

                           (identical? v val)
                           (present-path-segment k starting-index)))]
    (some * coll)))
