(ns non-idiomatic-parameter-binding)

(defn direct-optional [required & [optional]]
  [required optional])

(defn multiple-optional [required & [first-option second-option]]
  [required first-option second-option])

(defn- private-optional [required & [optional]]
  [required optional])

(defn valid-variadic [required & values]
  [required values])

(defn anonymous-wrapper []
  (fn [required & [optional]]
    [required optional]))

(def quoted-definition
  '(defn quoted [required & [optional]]
     [required optional]))

(defmacro generated-definition []
  `(defn generated [required & [optional]]
     [required optional]))

(let [defn (fn [& _] nil)]
  (defn shadowed [required & [optional]]
    [required optional]))
