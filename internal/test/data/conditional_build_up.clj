(ns conditional-buildup-let)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: single conditional assoc over same local map
(defn single-conditional-assoc []
  (let [m {}
        m (if true (assoc m :only 1) m)]
    m))

;; Example 2: two successive conditional assocs
(defn build-opts [x y]
  (let [m {}
        m (if x (assoc m :a 1) m)
        m (if y (assoc m :b 2) m)]
    m))

;; Example 3: three successive conditional assocs
(defn build-config [a b c]
  (let [cfg {:base true}
        cfg (if a (assoc cfg :a 1) cfg)
        cfg (if b (assoc cfg :b 2) cfg)
        cfg (if c (assoc cfg :c 3) cfg)]
    cfg))

;; Example 4: qualified if/assoc names
(defn qualified-core [x]
  (let [m {}
        m (clojure.core/if x
            (clojure.core/assoc m :a 1)
            m)]
    m))

;; Example 5: best run appears after an unrelated binding sequence
(defn later-run [p q]
  (let [x 1
        y 2
        m {}
        m (if p (assoc m :a 1) m)
        m (if q (assoc m :b 2) m)]
    m))

;; Example 6: run is broken later, but an earlier valid one-step run exists
(defn broken-run [p q]
  (let [m {}
        m (if p (assoc m :a 1) m)
        m :reset
        m (if q (assoc m :b 2) m)]
    m))

;; Example 7: destructuring appears in the let, but the detected run is on `m`
(defn destructuring-binding [p]
  (let [{:keys [a]} {:a 1}
        m {}
        m (if p (assoc m :x a) m)]
    m))

;; Example 8: repeated symbol later after another binding, but the first run already matches
(defn non-contiguous-run [p q]
  (let [m {}
        m (if p (assoc m :a 1) m)
        x 42
        m (if q (assoc m :b 2) m)]
    m))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 1: let with no repeated binding name
(defn simple-let []
  (let [a 1
        b 2]
    (+ a b)))

;; Example 2: same symbol rebound, but with non-matching operation
(defn merge-style []
  (let [m {}
        m (if :k (merge m {:a 1}) m)]
    m))

;; Example 3: same symbol rebound, but then branch is not assoc
(defn next-style [pred xs]
  (let [xs [1 2 3]
        xs (if pred (next xs) xs)]
    xs))

;; Example 4: else branch does not return the same symbol
(defn wrong-else [p]
  (let [m {}
        m (if p (assoc m :a 1) {})]
    m))

;; Example 5: first binding is already an if+assoc shape, so there is no valid base
(defn no-base-binding [p]
  (let [m (if p (assoc m :a 1) m)]
    m))

;; Example 6: still no valid base even with two repeated bindings
(defn no-valid-base-run [p q]
  (let [m (if p (assoc m :a 1) m)
        m (if q (assoc m :b 2) m)]
    m))

;; Example 7: repeated symbol, but only non-matching rebinding
(defn repeated-non-pattern [p]
  (let [m {}
        m (if p (update m :a inc) m)]
    m))

;; Example 8: destructuring as the target binding, not a simple symbol
(defn destructuring-target [p]
  (let [{:keys [m]} {:m {}}
        {:keys [m]} (if p {:m (assoc m :x 1)} {:m m})]
    m))