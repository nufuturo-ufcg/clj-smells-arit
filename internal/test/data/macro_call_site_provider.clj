(ns macro-call-site.provider)

(defmacro duplicated [expr]
  `(do ~expr ~expr))
