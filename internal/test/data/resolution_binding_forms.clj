(ns resolution-binding-forms)

(defn conditional-bindings [value]
  (if-let [bound value]
    bound
    value)
  (when-let [also-bound value]
    also-bound))

(defn loop-binding [value]
  (loop [current value]
    (if current
      current
      (recur value))))

(defn destructured-literals []
  (let [{:keys [items] :as record} {:items [1 2]}
        [first-item second-item] items]
    [items first-item second-item record]))

(defn destructured-unknown [source]
  (let [{:keys [items]} source]
    items))
