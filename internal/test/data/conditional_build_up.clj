(ns test.conditional-buildup-let)

;; =============================================================================
;; POSITIVE: series of (if pred (assoc sym ...) sym) — report (2+ conditional assocs)
;; =============================================================================

;; Map built with a *series* of conditional assocs — cond-> fits
(defn build-opts [x y]
  (let [m {}
        m (if x (assoc m :a 1) m)
        m (if y (assoc m :b 2) m)]
    m))

;; =============================================================================
;; NEGATIVE: no report
;; =============================================================================

;; Let with no repeated binding name
(defn simple-let []
  (let [a 1
        b 2]
    (+ a b)))

;; Same symbol rebound but only ONE (if pred (assoc ...) sym) — not a "series", no report
(defn with-doc [name doc macro-args]
  (let [attr (if (map? (first macro-args)) (first macro-args) {})
        attr (if doc (assoc attr :doc doc) attr)
        attr (if (meta name) (conj (meta name) attr) attr)]
    [(with-meta name attr) macro-args]))

;; Single conditional assoc — only one (if (assoc m ...) m), so no series, no report
(defn single-conditional-assoc []
  (let [m {}
        m (if true (assoc m :only 1) m)]
    m))

;; MERGE, not assoc — not matched
(defn merge-style []
  (let [m {}
        m (if :k (merge m {:a 1}) m)]
    m))

;; (next sym), not (assoc sym ...) — parsing, not build-up
(defn next-style [pred xs]
  (let [xs [1 2 3]
        xs (if pred (next xs) xs)]
    xs))
