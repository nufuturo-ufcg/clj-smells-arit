(ns test.file2)

(defn process-customer-data [customer]
  (let [name (clojure.string/trim (:name customer))
        email (clojure.string/lower-case (:email customer))
        age (:age customer)]
    (when (and (not (clojure.string/blank? name))
               (not (clojure.string/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

(defn handle-numbers [values]
  (let [filtered (filter pos? values)
        doubled (map #(* 2 %) filtered)
        total (reduce + doubled)]
    {:filtered filtered
     :doubled doubled
     :total total}))

(defn check-data [input]
  (when (and input (not-empty input))
    (every? number? input)))

(defn unique-function [x y]
  (+ x y (* x y))) 