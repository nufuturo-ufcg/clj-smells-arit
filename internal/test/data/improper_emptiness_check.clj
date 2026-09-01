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

;; A literal vector proves the collection contract locally.
(defn known-empty [xs]
  (= 0 (count [xs])))

(defn known-non-empty-condition [xs]
  (when (> (count [xs]) 0)
    :present))

(defn known-keys-condition [metadata]
  (when (> (count (keys metadata)) 0)
    :present))

(defn known-vals-condition [metadata]
  (when (> (count (vals metadata)) 0)
    :present))

(defn known-lazy-condition [items]
  (when (> (count (map inc items)) 0)
    :present))

(defn unresolved-producer-condition [items]
  (when (> (count (external-items items)) 0)
    :present))

(defn shadowed-empty [empty? items]
  (when-not (empty? items)
    :present))

;; An invalid core producer arity is not an executable collection expression.
(defn invalid-hash-map-arity [items]
  (when (> (count (hash-map :only)) 0)
    items))

;; An invalid arity must not inherit mapv's eager collection type.
(defn invalid-mapv-arity [items]
  (when (> (count (mapv inc)) 0)
    items))

;; Do not recommend a replacement that is shadowed by a local binding.
(defn shadowed-seq-replacement [seq items]
  (when-not (empty? items)
    :present))

(defn shadowed-empty-replacement [empty? items]
  (= 0 (count [items])))

(defn shadowed-seq-and-boolean-replacements [seq boolean items]
  (not (empty? items)))

;; nil is a valid collection input for count/empty?/seq.
(defn nil-count-condition []
  (when (> (count nil) 0)
    :present))
