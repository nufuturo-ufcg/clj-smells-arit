(ns blaze.terminology-service.local.code-system
  "Main code system functionality."
  (:refer-clojure :exclude [find list])
  (:require
   [blaze.anomaly :refer [if-ok when-ok]]
   [blaze.async.comp :as ac :refer [do-sync]]
   [blaze.db.api :as d]
   [blaze.fhir.util :as fu]
   [blaze.terminology-service.local.code-system :as-alias cs]
   [blaze.terminology-service.local.code-system.bcp-13]
   [blaze.terminology-service.local.code-system.bcp-47]
   [blaze.terminology-service.local.code-system.core :as c]
   [blaze.terminology-service.local.code-system.default]
   [blaze.terminology-service.local.code-system.sct]
   [blaze.terminology-service.local.code-system.ucum]
   [blaze.terminology-service.local.graph :as graph]
   [blaze.terminology-service.local.lookup :as lookup]
   [blaze.terminology-service.local.validate-code :as vc]
   [blaze.terminology-service.local.value-set.validate-code.issue :as issue]))

(defn- assoc-graph [{concepts :concept :as code-system}]
  (assoc code-system :default/graph (graph/build-graph concepts)))

(defn- find-in-tx-resources
  ([{:keys [tx-resources] ::cs/keys [required-content]
     :or {required-content #{"complete" "fragment"}}} url]
   (some
    (fn [{:fhir/keys [type] :as resource}]
      (when (identical? :fhir/CodeSystem type)
        (when (= url (:value (:url resource)))
          (when (required-content (:value (:content resource)))
            (ac/completed-future (assoc-graph resource))))))
    tx-resources))
  ([{:keys [tx-resources] ::cs/keys [required-content]
     :or {required-content #{"complete" "fragment"}}} url version]
   (some
    (fn [{:fhir/keys [type] :as resource}]
      (when (identical? :fhir/CodeSystem type)
        (when (= url (:value (:url resource)))
          (when (= version (:value (:version resource)))
            (when (required-content (:value (:content resource)))
              (ac/completed-future (assoc-graph resource)))))))
    tx-resources)))
