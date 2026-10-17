(ns macros-impl)

(defmacro real-debug [x]
  `(println "real-debug:" ~x))
