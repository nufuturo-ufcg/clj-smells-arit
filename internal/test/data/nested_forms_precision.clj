(ns nested-forms-precision)

;; Two lets remain below the default threshold of three.
(defn two-lets [x]
  (let [a (:a x)]
    (let [b (:b a)]
      b)))

;; An additional body expression makes flattening non-local.
(defn let-with-extra-expression [x]
  (let [a (:a x)]
    (println a)
    (let [b (:b a)]
      (let [c (:c b)]
        c))))

;; Legal nested shadowing would become duplicate bindings after flattening.
(defn shadowed-let [x]
  (let [value x]
    (let [value (inc value)]
      (let [result value]
        result))))

;; A tracked form in a binding initializer is not a body chain.
(defn let-in-initializer [x]
  (let [a (let [b x]
            (let [c b]
              c))]
    a))

;; Conditional chains do not have a proven local rewrite.
(defn conditional-chain [x]
  (when-let [a (:a x)]
    (when-let [b (:b a)]
      (when-let [c (:c b)]
        c))))

;; false and nil short-circuit differently under some->.
(defn false-sensitive [x]
  (if-let [a (:a x)]
    (if-let [b (:b a)]
      b
      :missing-b)
    :missing-a))

;; for flattening changes the result shape.
(defn nested-for [xs]
  (for [x xs]
    (for [y (:children x)]
      [x y])))

;; try is a semantic boundary.
(defn try-boundary [xs]
  (doseq [x xs]
    (try
      (doseq [y (:children x)]
        (println y))
      (catch Exception e
        (println e)))))

;; An effect between loops prevents vector concatenation.
(defn doseq-with-extra-expression [xs]
  (doseq [x xs]
    (println :outer x)
    (doseq [y (:children x)]
      (println y))))

;; Shadowing a generator would create a duplicate name in one vector.
(defn shadowed-doseq [xs]
  (doseq [x xs]
    (doseq [x (:children x)]
      (println x))))

;; Deferred and quoted forms are not executable direct chains.
(defn deferred-and-quoted [x]
  (let [a x]
    (delay
      (let [b a]
        (let [c b]
          c))))
  '(let [a 1] (let [b 2] (let [c 3] c))))
