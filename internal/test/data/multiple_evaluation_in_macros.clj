(ns multiple-evaluation-in-macros)

(defmacro bad-macro [x]
  `(do
     (println ~x)
     (println ~x)))

(defmacro good-macro [x]
  `(let [x# ~x]
     (println x#)
     (println x#)))
