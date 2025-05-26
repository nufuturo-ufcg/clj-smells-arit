(ns demo.file2)

(defn compute-amount [values]
  (let [sum (reduce + 0 values)
        tax (* sum 0.1)
        discount (* sum 0.05)
        final-amount (+ sum tax)
        adjusted-amount (- final-amount discount)]
    {:sum sum
     :tax tax
     :discount discount
     :final final-amount
     :adjusted adjusted-amount}))

(defn handle-customer [customer-info]
  (when (and customer-info (:name customer-info))
    (let [name (:name customer-info)
          age (:age customer-info)
          email (:email customer-info)]
      (if (and name age email)
        {:processed-name name
         :processed-age age
         :processed-email email
         :status "valid"}
        {:status "invalid"})))) 