(ns nested-forms)

(defn bad-nesting [x y]
  (let [a 1]
    (let [b 2]
      (+ a b x y))))

(defn bad-loops [xs ys]
  (doseq [x xs]
    (doseq [y ys]
      (println x y))))
