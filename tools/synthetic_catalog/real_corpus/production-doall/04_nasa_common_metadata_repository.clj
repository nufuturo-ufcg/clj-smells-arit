(ns cmr.metadata-db.data.oracle.search
  "Provides implementations of the cmr.metadata-db.data.concepts/ConceptStore protocol methods
  for retrieving concepts using parameters"
  (:require
   [clj-time.coerce :as time-coerce]
   [clojure.java.jdbc :as j]
   [clojure.set :as set]
   [clojure.string :as string]
   [cmr.common.log :refer (debug info warn error)]
   [cmr.common.concepts :as cc]
   [cmr.common.util :as util]
   [cmr.metadata-db.data.concepts :as c]
   [cmr.metadata-db.data.oracle.concept-tables :as tables]
   [cmr.metadata-db.data.oracle.concepts :as oc]
   [cmr.metadata-db.data.oracle.sql-helper :as sh]
   [cmr.oracle.sql-utils :as su :refer [insert values select from where with order-by desc delete as]])
  (:import
   (cmr.oracle.connection OracleStore)))

(def common-columns
  "A set of common columns for all concept types."
  #{:native_id :concept_id :revision_date :metadata :deleted :revision_id :format :transaction_id})

(def concept-type->columns
  "A map of concept type to the columns for that type in the database."
 (merge
  {:granule (into common-columns
                  [:provider_id :parent_collection_id :delete_time :granule_ur])
   :collection (into common-columns
                     [:provider_id :entry_title :entry_id :short_name :version_id :delete_time
                      :user_id])
   :tag (into common-columns [:user_id])
   :tag-association (into common-columns
                          [:associated_concept_id :associated_revision_id
                           :source_concept_identifier :user_id])
   :access-group (into common-columns [:provider_id :user_id])
   :service (into common-columns [:provider_id :service_name :user_id])
   :tool (into common-columns [:provider_id :tool_name :user_id])
   :acl (into common-columns [:provider_id :user_id :acl_identity])
   :humanizer (into common-columns [:user_id])
   :index-set (into common-columns [:user_id])
   :subscription (into common-columns
                       [:provider_id :subscription_name :subscriber_id
                        :collection_concept_id :user_id
                        :normalized_query :subscription_type :aws_arn])
   :variable (into common-columns [:provider_id :variable_name :measurement :user_id :fingerprint])
   :variable-association (into common-columns
                               [:associated_concept_id :associated_revision_id
                                :source_concept_identifier :user_id])
   :service-association (into common-columns
                              [:associated_concept_id :associated_revision_id
                               :source_concept_identifier :user_id])
   :tool-association (into common-columns
                           [:associated_concept_id :associated_revision_id
                            :source_concept_identifier :user_id])
   :generic-association (into common-columns
                               [:associated_concept_id :associated_revision_id
                                :source_concept_identifier :source_revision_id :user_id])}
   (zipmap (cc/get-generic-concept-types-array)
           (repeat (into common-columns [:provider_id :document_name :schema :user_id :created_at])))))

(defn find-associations
  "Find all associations in the database table that are part of a concept id
  that is passed in through params. The parameters look like the following:
  {:associated-concept-id \"C1200000013-PROV1\" :source-concept-identifier \"C1200000013-PROV1\"}"
  [db params]
  (let [fields (:generic-association concept-type->columns)
        table (tables/get-table-name nil :generic-association)
        stmt (su/build (select (vec fields)
                               (from table)
                               (when-not (empty? params)
                                 (where (sh/find-params->sql-clause params true)))))]
    (j/with-db-transaction
     [conn db]
     ;; doall is necessary to force result retrieval while inside transaction - otherwise
     ;; connection closed errors will occur
     (doall
      (mapv (fn [result]
              (oc/db-result->concept-map :generic-association conn "CMR" result))
            (su/query conn stmt))))))
