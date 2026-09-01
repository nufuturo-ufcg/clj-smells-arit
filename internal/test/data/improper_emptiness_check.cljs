(ns improper-emptiness-check-cljs)

;; JavaScript interop does not prove the concrete collection type, but this is
;; still a direct emptiness check consumed only as control-flow truthiness.
(defn process-input! [input]
  (when-not (empty? (.-value input))
    :present))

;; A public boolean result must not be changed to the sequence returned by seq.
(defn input-has-value? [input]
  (not (empty? (.-value input))))

;; Unknown numeric comparisons remain contextual because the input contract is
;; not available from this file.
(defn input-length-positive? [input]
  (> (count (.-value input)) 0))
