(ns case-with-non-literal-test-values)

(defn literal-constants [value]
  (case value
    :ready :keyword
    (ok error) :grouped-symbols
    true :boolean
    nil :nil
    0 :number
    :default))

(def global-constant :ready)

(defn global-symbol [value]
  (case value
    global-constant :global-symbol
    :default))

(defn grouped-local-symbol [value]
  (let [expected :ready]
    (case value
      (expected) :literal-group
      :default)))

(defn local-is-evaluated [value]
  (let [expected :ready]
    (case value
      expected :local-value
      :default)))

(defn structured-literal-constants [value]
  (case value
    #example/tag "ready" :tagged
    (`literal-target) :syntax-quoted
    [:a :b] :vector
    {:kind :ok} :map
    #{:a :b} :set
    :default))

(defmacro generated-case [value]
  `(case ~value
     runtime-symbol :generated
     :default))

(defn shadowed-case [case value]
  (case value
    runtime-value :local-function
    :default))
