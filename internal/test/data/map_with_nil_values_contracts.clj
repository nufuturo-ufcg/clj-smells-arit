(ns map-with-nil-values-contracts)

;; These are legitimate data contracts that require distinguishing nil from
;; an absent key. The detector must keep them contextual.
(defn json-payload [body]
  {:body body
   :error nil})

(defn patch-payload [entity]
  (assoc entity :display-name nil))

(def ast-node
  {:op :literal
   :form nil})

(def fixed-schema
  {:value nil
   :present? true})
