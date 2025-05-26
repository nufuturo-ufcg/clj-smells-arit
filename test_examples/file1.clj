(ns test.file1)

(defn process-user-data [user]
  (let [name (clojure.string/trim (:name user))
        email (clojure.string/lower-case (:email user))
        age (:age user)]
    (when (and (not (clojure.string/blank? name))
               (not (clojure.string/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

(defn process-data [items]
  (let [filtered (filter pos? items)
        doubled (map #(* 2 %) filtered)
        total (reduce + doubled)]
    {:filtered filtered
     :doubled doubled
     :total total}))

(defn validate-input [data]
  (when (and data (not-empty data))
    (every? number? data))) 