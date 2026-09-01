(ns non-idiomatic-parameter-binding-multi-arity)

(defn multi-arity
  ([required]
   required)
  ([required & [optional]]
   [required optional]))

(defn all-idiomatic
  ([required]
   required)
  ([required optional]
   [required optional]))

(def named-fn
  (fn named
    ([required]
     required)
    ([required & [optional]]
     [required optional])))
