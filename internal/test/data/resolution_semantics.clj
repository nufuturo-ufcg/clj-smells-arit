(ns resolution-semantics
  (:require [clojure.string :as str :refer [blank?]]))

(defn alias-and-refer [value]
  [(str/trim value) (blank? value)])

(defn shadowed-operation [trim]
  (trim "value"))

(defn sequential-bindings [earlier]
  (let [later earlier
        earlier :inner]
    [later earlier]))

(defn destructured-bindings [{:keys [name] :as user} [first-value second-value & rest-values]]
  [name user first-value second-value rest-values])

(defn non-evaluated-forms []
  '(missing-call)
  `(+ ~missing-value))
