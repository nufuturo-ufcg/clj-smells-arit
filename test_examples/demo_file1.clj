(ns demo.file1)

(defn calculate-total [items]
  (let [sum (reduce + 0 items)
        tax (* sum 0.1)
        discount (* sum 0.05)
        final-amount (+ sum tax)
        adjusted-amount (- final-amount discount)]
    {:sum sum
     :tax tax
     :discount discount
     :final final-amount
     :adjusted adjusted-amount}))

(defn process-user [user-data]
  (when (and user-data (:name user-data))
    (let [name (:name user-data)
          age (:age user-data)
          email (:email user-data)]
      (if (and name age email)
        {:processed-name name
         :processed-age age
         :processed-email email
         :status "valid"}
        {:status "invalid"})))) 