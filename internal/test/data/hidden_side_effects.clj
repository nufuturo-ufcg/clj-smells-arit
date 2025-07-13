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
      (cons (first s) #_{:clj-kondo/ignore [:invalid-arity]}
                      (lazy-show-nums)))))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; = EXAMPLES WITH '!' OR KEYWORDS =

;; Example 1: 'map' with println using '!'
(defn greet-user! [user]
  (println "Hello," (:name user)))

(defn greet-users! [users]
  (doseq [user users]
    (greet-user! user)))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users! users))

;; Example 2: 'reduce' with side-effect using key-word 'log'
(declare sum)
(defn log-and-accumulate [acc x]
  (println "Adding" x)
  (+ acc x))

(defn sum-with-logging [nums]
  (reduce accumulate 0 nums))

;; = EXAMPLES WITH REFACTORING =

;; Example 3: 'map' without side effects
(defn greet-user [user]
  (str "Greeted " (:name user)))

(defn greet-users [users]
  (map greet-user users))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users users))

;; Example 4: 'reduce' without side-effects
(declare sum)
(defn accumulate [acc x]
  (+ acc x))

(defn sum [nums]
  (reduce accumulate 0 nums))

;; Example 5: 'filter without side-effects
(declare filtered-values)
(defn is-even [x]
  (even? x))

(defn filtered-values [xs]
  (filter is-even xs))

;; Example 6: 'lazy-seq' without side effects
(defn lazy-numbers []
  (range 5))

;; Example 7: 'filter' without side effects
(declare filter-evens)
(defn is-even [x]
  (even? x))

(defn filter-evens [nums]
  (filter is-even nums))

;;Example 8: for without-side effects
(declare amplify-all)
(defn scale [x]
  (* 2 x))

(defn amplify-all [nums]
  (for [n nums]
    (scale n)))

;; Example 9: 'comp' without side-effects
(declare process-nums)
(defn raise [x]
  (inc x))

(def amplify-after-raise (comp #(* 2 %) raise))

(defn process-nums [nums]
  (map amplify-after-raise nums))

;; Example 10: 'lazy-seq' without side-effects
(defn lazy-show-nums [s]
  (lazy-seq
    (when-let [s (seq s)]
      (cons (first s) (lazy-show-nums (rest s))))))

