(ns test-macro-extended)

(defn some-thread [value f]
  (some-> value
          (f 1)
          vec))

(defn some-thread-last [value f]
  (some->> value
           (map f)
           vec))

(defn conditional-thread [value f enabled?]
  (cond-> value
    enabled? (f 1)
    (not enabled?) vec))

(defn conditional-thread-last [value f enabled?]
  (cond->> value
    enabled? (map f)
    (not enabled?) vec))

(defn invalid-conditional-thread [value enabled?]
  (cond-> value enabled? (identity 1) false))

(def quoted-thread '(some-> value identity))
