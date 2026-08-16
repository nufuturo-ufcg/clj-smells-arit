(ns nested-forms)

(defn flattenable-lets [input]
  (let [a (:a input)]
    (let [{:keys [b]} a]
      (let [^String c (str b)]
        (.trim c)))))

(defn flattenable-doseqs [xs]
  (doseq [x xs]
    (doseq [y (:children x)]
      (println x y))))

(defn flattenable-doseqs-with-modifiers [xs]
  (doseq [x xs
          :when (:active x)]
    (doseq [y (:children x)
            :let [label (str (:id x) ":" (:id y))]]
      (println label))))
