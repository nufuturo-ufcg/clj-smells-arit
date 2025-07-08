(ns hidden-side-effects)

;; Example 1: 'map' with println
(defn greet-user [user]

  (println "Hello," (:name user))
  (str "Greeted " (:name user)))

(defn greet-users [users]
  (map greet-user users))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users users))

;; Example 2: 'reduce' with side-effect (not detecting)
(declare sum)
(defn accumulate [acc x]
  (println "Adding" x)
  (+ acc x))

(defn sum [nums]
  (reduce accumulate 0 nums))

;; Example 3: 'filter' with side effect
(declare filtered-values)
(defn side-effect-check [x]
  (println "Checking" x)
  (even? x))

(defn filtered-values [xs]
  (filter side-effect-check xs))

;; Example 4: 'lazy-seq' with a println
(defn lazy-numbers []
  (map #(do (println "Yielding" %) %) (range 5)))

;; Example 5: 'filter' with a println
(declare filter-evens)
(defn check-even [x]
  (println "Checking" x)
  (even? x))

(defn filter-evens [nums]
  (filter check-even nums))

;; Example 6: 'for' with side effect (not detected)
(declare amplify-all)
(defn show-and-scale [x]
  (println "Doubling" x)  ;; efeito colateral oculto
  (* 2 x))

(defn amplify-all [nums]
  (for [n nums]
    (show-and-scale n)))


;; Example 7: 'comp' with a printing function (not detected)
(declare process-nums)
(defn track-and-inc [x]
  (println "Incrementing" x) 
  (inc x))

(def inc-then-double (comp #(* 2 %) track-and-inc))

(defn process-nums [nums]
  (map inc-then-double nums)) 

;; Example 8: 'lazy-seq' with side-effect (not detected)
(defn lazy-show-nums [s]
  (lazy-seq
    (when-let [s (seq s)]
      (println "Yielding" (first s))
      (cons (first s) (lazy-show-nums)))))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

(defn greet-user! [user]
  ;; Side effect now explicit and named
  (println "Hello," (:name user)))

(defn greet-users! [users]
  ;; Use doseq for side effects
  (doseq [user users]
    (greet-user! user)))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users! users))
