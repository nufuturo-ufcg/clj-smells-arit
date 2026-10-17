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

(def association-concept-type->generic-association
  "Mapping of various association concept types to columns needed for migration to CMR_ASSOCIATIONS table."
  {:service-association {:association_type "SERVICE-COLLECTION" :kebab-key-mapping {:service-concept-id :source-concept-identifier}}
   :tag-association {:association_type "TAG-COLLECTION" :kebab-key-mapping {:tag-key :source-concept-identifier}}
   :tool-association {:association_type "TOOL-COLLECTION" :kebab-key-mapping {:tool-concept-id :source-concept-identifier}}
   :variable-association {:association_type "VARIABLE-COLLECTION" :kebab-key-mapping {:variable-concept-id :source-concept-identifier}}})

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

(def single-table-with-providers-concept-type?
  "The set of concept types that are stored in a single table with a provider column. These concept
   types must include the provider id as part of the sql params"
  #{:access-group :variable :service :tool :subscription})

(defn columns-for-find-concept
  "Returns the table columns that should be included in a find-concept sql query"
  [concept-type params]
  (let [exclude-metadata? (= "true" (:exclude-metadata params))
        all-fields (concept-type->columns concept-type)]
    (disj all-fields (when exclude-metadata? :metadata))))

(defn- params->sql-params
  "Converts the search params into params that can be converted into a sql condition clause."
  [concept-type providers params]
  (if (or (every? :small providers)
          (single-table-with-providers-concept-type? concept-type)
          (cc/generic-concept? concept-type))
    (dissoc params :concept-type :exclude-metadata)
    (dissoc params :concept-type :exclude-metadata :provider-id)))

(defmulti gen-find-concepts-in-table-sql
  "Create the SQL for the given params and table.
  If :include-all is true, all revisions of the concepts will be returned. This is needed for the
  find-latest-concepts function to later filter out the latest concepts that satisfy the search in memory."
  (fn [concept-type table fields params]
    (condp = concept-type
      :granule :granule-search
      concept-type)))

(defmethod gen-find-concepts-in-table-sql :granule-search
  [concept-type table fields params]
  (let [stmt (gen-find-concepts-in-table-sql :dummy table fields params)]
    ;; add index hint to the generated sql statement
    (update-in stmt [0] #(string/replace % #"SELECT" (format "SELECT /*+ INDEX(%s) */" table)))))

(defmethod gen-find-concepts-in-table-sql :default
  [concept-type table fields params]
  (if (:include-all params)
    (let [params (dissoc params :include-all)]
      (su/build (select (vec fields)
                        (from table)
                        (where `(in :concept_id
                                    ~(select [:concept_id]
                                             (from table)
                                             (when-not (empty? params)
                                               (where (sh/find-params->sql-clause params)))))))))
    (su/build (select (vec fields)
                      (from table)
                      (when-not (empty? params)
                        (where (sh/find-params->sql-clause params)))))))

(defmethod gen-find-concepts-in-table-sql :subscription
  [_concept-type table fields params]
  (let [fields (cons :last_notified_at fields)
        root-query (gen-find-concepts-in-table-sql :default table fields params)
        updated-query (string/replace-first
                       (first root-query)
                       #"WHERE"
                       (str " LEFT JOIN cmr_sub_notifications"
                            " ON cmr_subscriptions.concept_id = cmr_sub_notifications.subscription_concept_id"
                            " WHERE"))]
    (cons updated-query (rest root-query))))

(defmulti find-concepts-in-table
  "Retrieve concept maps from the given table, handling small providers separately from
  normal providers."
  (fn [db table concept-type providers params]
    (or (:small (first providers))
        (= :variable concept-type)
        (= :service concept-type)
        (= :tool concept-type)
        (= :subscription concept-type)
        (cc/generic-concept? concept-type))))

(defmethod find-concepts-in-table true
  [db table concept-type providers params]
  (let [fields (columns-for-find-concept concept-type params)
        params (params->sql-params concept-type
                                   providers
                                   (assoc params :provider-id (map :provider-id providers)))
        stmt (gen-find-concepts-in-table-sql concept-type table fields params)]
    (j/with-db-transaction
      [conn db]
      (doall
        (mapv #(oc/db-result->concept-map concept-type conn (:provider_id %) %)
              (su/query conn stmt))))))

(defmethod find-concepts-in-table :default
  [db table concept-type providers params]
  {:pre [(or (single-table-with-providers-concept-type? concept-type)
             (== 1 (count providers)))]}
  (let [provider-ids (map :provider-id providers)
        fields (disj (columns-for-find-concept concept-type params) :provider_id)
        params (params->sql-params concept-type providers (assoc params :provider-id provider-ids))
        params (if (or (= :variable-association concept-type)
                       (= :tag-association concept-type)
                       (= :service-association concept-type)
                       (= :tool-association concept-type))
                 (-> params
                     (set/rename-keys
                      (get-in association-concept-type->generic-association [concept-type :kebab-key-mapping]))
                     (assoc :association-type
                            (get-in association-concept-type->generic-association [concept-type :association_type])))
                 params)
        stmt (gen-find-concepts-in-table-sql concept-type table fields params)]
    (j/with-db-transaction
      [conn db]
      ;; doall is necessary to force result retrieval while inside transaction - otherwise
      ;; connection closed errors will occur
      (doall (mapv (fn [result]
                    (oc/db-result->concept-map concept-type conn
                                               (or (:provider_id result)
                                                   (:provider-id (first providers)))
                                               result))
                   (su/query conn stmt))))))

(defn find-latest-concepts
  [db provider params]
  {:pre [(:concept-type params)]}
  ;; First we find all revisions of the concepts that have at least one revision that matches the
  ;; search parameters. Then we find the latest revisions of those concepts and match with the
  ;; search parameters again in memory to find what we are looking for.
  (let [concept-type (:concept-type params)
        table (tables/get-table-name provider concept-type)
        revision-concepts (find-concepts-in-table db table concept-type [provider]
                                                  (assoc params :include-all true))
        latest-concepts (->> revision-concepts
                             (group-by :concept-id)
                             (map (fn [[concept-id concepts]]
                                    (->> concepts (sort-by :revision-id) reverse first))))]
    (c/search-with-params latest-concepts params)))
