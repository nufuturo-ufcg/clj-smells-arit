(ns case-with-non-literal-global-var)

(def status :ready)
(defonce mode :safe)

(defn uses-status [value]
  (case value
    status :ready
    :other))

(defn uses-mode [value]
  (case value
    mode :safe
    :other))

;; These remain silent: a quoted symbol and an unresolved symbol can be
;; intentional literal constants, so the AST cannot establish misuse.
(defn literal-symbol [value]
  (case value
    'status :symbol
    :other))

(defn unresolved-symbol [value]
  (case value
    external-symbol :symbol
    :other))
