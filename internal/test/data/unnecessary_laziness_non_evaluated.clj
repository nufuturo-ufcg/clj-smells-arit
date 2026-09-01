(ns unnecessary-laziness-non-evaluated)

(def quoted-map
  '(vec (map inc xs)))

(def syntax-quoted-map
  `(vec (map inc xs)))

;; (vec (map inc xs))

(def discarded-map
  #_(vec (map inc xs)))
