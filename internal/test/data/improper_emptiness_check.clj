(ns improper-emptiness-check)

(defn empty-return [xs]
  (= 0 (count xs)))

(defn non-empty-condition [xs]
  (when (> (count xs) 0)
    :present))

;; seq would change the return type from boolean to a sequence.
(defn non-empty-return [xs]
  (> (count xs) 0))

(defn negated-condition [xs]
  (if (not (empty? xs))
    :present
    :absent))

;; Same reason: preserve the explicit boolean API.
(defn negated-return [xs]
  (not (empty? xs)))

(defn direct-when-not [xs]
  (when-not (empty? xs)
    :present))

(defn shadowed-count [count xs]
  (= 0 (count xs)))
