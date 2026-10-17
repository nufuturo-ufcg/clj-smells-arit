(ns improper-emptiness-lazy-count)

;; count realizes a lazy sequence completely; seq and empty? realize at most
;; the first chunk, so these rewrites are not proven.
(defn lazy-sources [xs f]
  (let [kept (filter odd? xs)
        mapped (map f xs)]
    [(when (> (count kept) 0) :some)
     (when (pos? (count mapped)) :some)
     (when (= 0 (count (remove nil? xs))) :none)]))

;; A realized vector keeps the proven rewrite.
(defn eager-source [xs]
  (let [v (vec xs)]
    (when (> (count v) 0) :some)))

;; empty? and seq realize the same prefix of a lazy sequence.
(defn lazy-empty-check [xs]
  (let [kept (filter odd? xs)]
    (when-not (empty? kept) :some)))
