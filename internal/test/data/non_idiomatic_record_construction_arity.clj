(ns non-idiomatic-record-construction-arity)

(defrecord User [id name])

(defn invalid-constructions []
  (User. 1)
  (new User 1 "Ana" :extra))

(defn valid-construction []
  (User. 1 "Ana"))
