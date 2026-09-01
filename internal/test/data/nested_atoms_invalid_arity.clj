(ns nested-atoms-invalid-arity)

;; An invalid inner creator cannot be evidence of a nested managed reference.
(def invalid (atom (atom)))
