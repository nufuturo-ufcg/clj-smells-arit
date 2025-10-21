(ns circular.test)

(defn a []
  (b))

(defn b []
  (a))

