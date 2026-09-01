(ns unnecessary-macro-call-site.provider)

(defmacro add-one [x]
  `(+ ~x 1))
