(ns hidden-side-effects)

;; Example 1: 
(declare greet-users)
(defn greet-user [user]

  (println "Hello," (:name user))
  (str "Greeted " (:name user)))

(defn greet-users [users]
  (map greet-user users))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users users))

;; Example 2:
(declare sum-with-logging)
(defn log-and-accumulate [acc x]
  (println "Adding" x)
  (+ acc x))

(defn sum-with-logging [nums]
  (reduce log-and-accumulate 0 nums))

(declare filtered-values)
(defn side-effect-check [x]
  (println "Checking" x)
  (even? x))

(defn filtered-values [xs]
  (filter side-effect-check xs))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========
(declare greet-users!)
(defn greet-user! [user]
  ;; Side effect now explicit and named
  (println "Hello," (:name user)))

(defn greet-users! [users]
  ;; Use doseq for side effects
  (doseq [user users]
    (greet-user! user)))

(let [users [{:name "Alice"} {:name "Bob"} {:name "Carol"}]]
  (greet-users! users))
