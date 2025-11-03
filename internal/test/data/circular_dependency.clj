(ns circular.test)

(defn a []
  (b))

(defn b []
  (a))

(defn c []
  (d))

(defn d []
  (c))
