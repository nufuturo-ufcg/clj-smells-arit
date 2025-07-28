;; ============================================================================
;; REDUNDANT DO BLOCK - Code Smell Examples
;; ============================================================================
;; This file demonstrates classic examples of redundant `do` blocks in Clojure
;; and their proper refactored versions. Based on idiomatic Clojure practices.

;; ----------------------------------------------------------------------------
;; CLASSIC REDUNDANT DO BLOCK EXAMPLES (Code Smells)
;; ----------------------------------------------------------------------------

;; Example 1: Redundant do in let body
(defn process-user-data [user]
  (let [name (:name user)
        email (:email user)]
    (do
      (println "Processing user:" name)
      (str name " <" email ">"))))

;; Example 2: Redundant do in when body
(defn validate-age [age]
  (when (> age 18)
    (do
      (println "User is adult")
      (println "Access granted")
      true)))

;; Example 3: Redundant do in if branches
(defn handle-response [status]
  (if (= status 200)
    (do
      (println "Success")
      :ok)
    (do
      (println "Error occurred")
      :error)))

;; Example 4: Redundant do in defn body (single expression)
(defn calculate-total [items]
  (do
    (reduce + (map :price items))))

;; Example 5: Redundant do in try/catch blocks
(defn safe-divide [a b]
  (try
    (do
      (println "Attempting division")
      (/ a b))
    (catch ArithmeticException e
      (do
        (println "Division by zero!")
        0))))

;; Example 6: Redundant do in doseq body
(defn print-users [users]
  (doseq [user users]
    (do
      (println "Name:" (:name user))
      (println "Email:" (:email user)))))

;; Example 7: Redundant do in loop/recur
(defn countdown [n]
  (loop [i n]
    (if (pos? i)
      (do
        (println "Count:" i)
        (recur (dec i)))
      (do
        (println "Done!")
        :finished))))

;; ----------------------------------------------------------------------------
;; PROPERLY REFACTORED VERSIONS (Code Smell Fixes)
;; ----------------------------------------------------------------------------

;; Example 1 Refactored: let body has implicit do
(defn process-user-data-fixed [user]
  (let [name (:name user)
        email (:email user)]
    (println "Processing user:" name)
    (str name " <" email ">")))

;; Example 2 Refactored: when body has implicit do  
(defn validate-age-fixed [age]
  (when (> age 18)
    (println "User is adult")
    (println "Access granted")
    true))

;; Example 3 Refactored: Use cond for multiple branches or extract functions
(defn handle-response-fixed [status]
  (cond
    (= status 200) (do
                     (println "Success")
                     :ok)
    :else          (do
                     (println "Error occurred")
                     :error)))

;; Even better: Extract to functions (most idiomatic)
(defn handle-response-best [status]
  (letfn [(success-response []
            (println "Success")
            :ok)
          (error-response []
            (println "Error occurred")
            :error)]
    (if (= status 200)
      (success-response)
      (error-response))))

;; Example 4 Refactored: Single expression doesn't need do
(defn calculate-total-fixed [items]
  (reduce + (map :price items)))

;; Example 5 Refactored: try/catch bodies have implicit do
(defn safe-divide-fixed [a b]
  (try
    (println "Attempting division")
    (/ a b)
    (catch ArithmeticException e
      (println "Division by zero!")
      0)))

;; Example 6 Refactored: doseq body has implicit do
(defn print-users-fixed [users]
  (doseq [user users]
    (println "Name:" (:name user))
    (println "Email:" (:email user))))

;; Example 7 Refactored: Extract functions for clarity
(defn countdown-fixed [n]
  (letfn [(print-count [i]
            (println "Count:" i)
            (recur (dec i)))
          (finish []
            (println "Done!")
            :finished)]
    (loop [i n]
      (if (pos? i)
        (print-count i)
        (finish)))))

;; Alternative: Use threading macros for pipeline operations
(defn process-data-pipeline [data]
  (->> data
       (filter :active)
       (map :value)
       (reduce +)))

;; Instead of nested do blocks:
;; (defn process-data-bad [data]
;;   (let [filtered (filter :active data)]
;;     (do
;;       (let [values (map :value filtered)]
;;         (do
;;           (reduce + values)))))) 