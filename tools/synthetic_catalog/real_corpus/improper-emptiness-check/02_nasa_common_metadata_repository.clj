(ns cmr.metadata-db.services.concept-service
  "Services to support the business logic of the metadata db."
  (:require
   [clj-time.core :as t]
   [clojure.set :as set]
   [cmr.common.api.context :as cmn-context]
   [cmr.common.concepts :as cu]
   [cmr.common.config :as cfg :refer [defconfig]]
   [cmr.common.date-time-parser :as p]
   [cmr.common.log :refer (debug info warn trace)]
   [cmr.common.mime-types :as mt]
   [cmr.common.services.errors :as errors]
   [cmr.common.services.messages :as cmsg]
   [cmr.common.time-keeper :as time-keeper]
   [cmr.common.util :as cutil]
   [cmr.metadata-db.data.concepts :as c]
   [cmr.metadata-db.data.ingest-events :as ingest-events]
   [cmr.metadata-db.data.providers :as provider-db]
   [cmr.metadata-db.services.concept-constraints :as cc]
   [cmr.metadata-db.services.concept-validations :as cv]
   [cmr.metadata-db.services.messages :as msg]
   [cmr.metadata-db.services.provider-service :as provider-service]
   [cmr.metadata-db.services.provider-validation :as pv]
   [cmr.metadata-db.services.search-service :as search]
   [cmr.metadata-db.services.util :as util])
  ;; Required to get code loaded
  ;; XXX This is really awful, and we do it a lot in the CMR. What we've got
  ;;     as a result of this is a leak from implementation into a separate part
  ;;     of the code ... not to mention that whatever is happing is is 100%
  ;;     implicit and chock-a-block with side effects. I believe the reason we
  ;;     do this is to work around issues with multimethods. We really need to
  ;;     refactor our multimethod code -- and this isn't the only reason
  ;;     (multimethods use slower code under the hood). If we really do need
  ;;     the flexible polymorphism that multimethods provide, then let's
  ;;     rethink our code reuse strategy around multimethods.
  (:require
   [cmr.metadata-db.data.oracle.concepts.acl]
   [cmr.metadata-db.data.oracle.concepts.collection]
   [cmr.metadata-db.data.oracle.concepts.generic-association]
   [cmr.metadata-db.data.oracle.concepts.generic-documents]
   [cmr.metadata-db.data.oracle.concepts.granule]
   [cmr.metadata-db.data.oracle.concepts.group]
   [cmr.metadata-db.data.oracle.concepts.humanizer]
   [cmr.metadata-db.data.oracle.concepts.index-set]
   [cmr.metadata-db.data.oracle.concepts.subscription]
   [cmr.metadata-db.data.oracle.concepts.service-association]
   [cmr.metadata-db.data.oracle.concepts.service]
   [cmr.metadata-db.data.oracle.concepts.tag-association]
   [cmr.metadata-db.data.oracle.concepts.tag]
   [cmr.metadata-db.data.oracle.concepts.tool-association]
   [cmr.metadata-db.data.oracle.concepts.tool]
   [cmr.metadata-db.data.oracle.concepts.variable-association]
   [cmr.metadata-db.data.oracle.concepts.variable]
   [cmr.metadata-db.data.oracle.concepts]
   [cmr.metadata-db.data.oracle.providers]
   [cmr.metadata-db.data.oracle.search]))

(defn- publish-tool-associations-update-event
  "Publish one concept-update-event for all non-tombstoned tool associations for the
  given tool concept; This is to trigger the reindexing of the associated collections in elastic
  search when tool is updated because tool info is indexed into the associated collections.
  Does nothing if the given concept is not a tool concept."
  [context concept-type concept-id]
  (when (= :tool concept-type)
    (let [search-params (cutil/remove-nil-keys
                         {:concept-type :tool-association
                          :tool-concept-id concept-id
                          :exclude-metadata true
                          :latest true})
          associations (filter #(= false (:deleted %))
                               (search/find-concepts context search-params))]
      (when (> (count associations) 0)
        (ingest-events/publish-event
         context
         (ingest-events/associations-update-event associations))))))
