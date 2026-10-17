(ns unknown-form-arguments
  (:require [labrepl.util :refer [code]]
            [kibit.rules.util :refer [defrules]]))

;; A display macro of another namespace may quote its arguments.
(defn page []
  [(code (def me {:first-name "Stu"}))
   (code (use 'student.dialect))])

;; A def in an ordinary function body still redefines a Var at call time.
(defn store! [x]
  (def last-x x))

;; Rule patterns given to an unknown top-level macro are data.
(defrules rules
  [(loop ?binding (do . ?exprs)) (loop ?binding . ?exprs)])

;; Record literals keep every base field.
(defrecord Node [v l r])
(def leaf #unknown_form_arguments.Node{:v 1 :l nil :r nil})
