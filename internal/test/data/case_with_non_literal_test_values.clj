(ns case-with-non-literal-test-values)

(defn literal-constants [value]
  (case value
    :ready :keyword
    (ok error) :grouped-symbols
    true :boolean
    nil :nil
    0 :number
    :default))

(defn local-is-evaluated [value]
  (let [expected :ready]
    (case value
      expected :local-value
      :default)))

(defn structured-literal-constants [value]
  (case value
    #example/tag "ready" :tagged
    [:a :b] :vector
    {:kind :ok} :map
    #{:a :b} :set
    :default))

(defmacro generated-case [value]
  `(case ~value
     runtime-symbol :generated
     :default))
