(ns verbose-checks)

(defn checks [x]
  (if (= 0 x)
    (println "zero"))
  (if (= x true)
    (println "true"))
  (if (= nil x)
    (println "nil"))
  (+ 1 x))
