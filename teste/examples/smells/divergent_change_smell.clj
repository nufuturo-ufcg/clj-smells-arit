(ns examples.smells.divergent-change)

(defn process-user [user]
  (let [full-name (str (:first-name user) " " (:last-name user)) ; DATA_TRANSFORMATION (str)
        valid-age? (>= (:age user) 18)]                         ; DATA_TRANSFORMATION (>=) (ou VALIDATION)
    (if valid-age?
      (do
        (println (str "User " full-name " is valid. Sending notification...")) ; IO_EFFECT (println), DATA_TRANSFORMATION (str)
        {:full-name full-name :status "Valid"})
      (do
        (println (str "User " full-name " is not valid. No notification sent.")) ; IO_EFFECT (println), DATA_TRANSFORMATION (str)
        {:full-name full-name :status "Invalid"}))))

(process-user {:first-name "Alice" :last-name "Smith" :age 22})
(process-user {:first-name "Bob" :last-name "Johnson" :age 17}) 