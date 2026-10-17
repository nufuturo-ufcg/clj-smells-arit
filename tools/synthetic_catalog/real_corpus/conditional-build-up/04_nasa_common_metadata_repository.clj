(ns cmr.search.services.query-service
  "Defines operations for querying for concepts.

  Steps to process a query:

  - Validate parameters
  - Convert parameters into query model
  - Query validation
  - Apply ACLs to query
  - Query simplification
  - Convert Query into Elastic Query Model
  - Send query to Elasticsearch
  - Convert query results into requested format"
  (:require
   [cheshire.core :as json]
   [clojure.set :as set]
   [cmr.elastic-utils.config :as es-config]
   [cmr.elastic-utils.search.es-wrapper :as esq]
   [cmr.common-app.services.search :as common-search]
   [cmr.common.concepts :as cc]
   [cmr.common.log :refer (info)]
   [cmr.common.mime-types :as mt]
   [cmr.common.services.errors :as errors]
   [cmr.common.services.search.query-model :as qm]
   [cmr.common.util :as u]
   [cmr.elastic-utils.es-helper :as es-helper]
   [cmr.elastic-utils.search.es-group-query-conditions :as gc]
   [cmr.elastic-utils.search.es-index :as common-idx]
   [cmr.elastic-utils.search.es-params-converter :as common-params]
   [cmr.elastic-utils.search.query-execution :as qe]
   [cmr.search.api.core :refer [log-search-result-metadata]]
   [cmr.search.data.elastic-search-index :as idx]
   [cmr.search.data.metadata-retrieval.metadata-cache :as metadata-cache]
   [cmr.search.results-handlers.provider-holdings :as ph]
   [cmr.search.services.aql.conversion :as a]
   [cmr.search.services.json-parameters.conversion :as jp]
   [cmr.search.services.parameters.conversion :as p]
   [cmr.search.services.parameters.legacy-parameters :as lp]
   [cmr.search.services.parameters.parameter-validation :as pv]
   [cmr.search.services.parameters.provider-short-name :as psn]
   [cmr.search.services.result-format-helper :as rfh]
   [cmr.spatial.codec :as spatial-codec]
   [cmr.spatial.tile :as tile])
  ;; These must be required here to make multimethod implementations available.
  ;; XXX This is not a good pattern for large software systems; we need to
  ;;     find a different way to accomplish this goal ... possibly use protocols
  ;;     instead.
  (:require
   cmr.search.data.complex-to-simple-converters.attribute
   cmr.search.data.complex-to-simple-converters.has-granules
   cmr.search.data.complex-to-simple-converters.has-granules-or-cwic
   cmr.search.data.complex-to-simple-converters.orbit
   cmr.search.data.complex-to-simple-converters.spatial
   cmr.search.data.complex-to-simple-converters.temporal
   cmr.search.data.complex-to-simple-converters.two-d-coordinate-system
   cmr.elastic-utils.search.elastic-results-to-query-results
   cmr.search.services.aql.converters.attribute
   cmr.search.services.aql.converters.attribute-name
   cmr.search.services.aql.converters.science-keywords
   cmr.search.services.aql.converters.spatial
   cmr.search.services.aql.converters.temporal
   cmr.search.services.aql.converters.two-d-coordinate-system
   cmr.search.services.json-parameters.converters.attribute
   cmr.search.services.parameters.converters.attribute
   cmr.search.services.parameters.converters.collection-query
   cmr.search.services.parameters.converters.equator-crossing-date
   cmr.search.services.parameters.converters.equator-crossing-longitude
   cmr.search.services.parameters.converters.humanizer
   cmr.search.services.parameters.converters.orbit-number
   cmr.search.services.parameters.converters.pass
   cmr.search.services.parameters.converters.platform
   cmr.search.services.parameters.converters.science-keyword
   cmr.search.services.parameters.converters.spatial
   cmr.search.services.parameters.converters.shapefile
   cmr.search.services.parameters.converters.geometry
   cmr.search.services.parameters.converters.temporal
   cmr.search.services.parameters.converters.temporal-facet
   cmr.search.services.parameters.converters.two-d-coordinate-system
   cmr.search.services.query-execution
   cmr.search.services.query-execution.facets.collection-v2-facets
   cmr.search.services.query-execution.facets.facets-results-feature
   cmr.search.services.query-execution.facets.facets-v2-results-feature
   cmr.search.services.query-execution.facets.granule-v2-facets
   cmr.search.services.query-execution.granule-counts-results-feature
   cmr.search.services.query-execution.has-granules-created-at-feature
   cmr.search.services.query-execution.has-granules-results-feature
   cmr.search.services.query-execution.has-granules-or-cwic-results-feature
   cmr.search.services.query-execution.has-granules-revised-at-feature
   cmr.search.services.query-execution.highlight-results-feature
   cmr.search.services.query-execution.tags-results-feature
   cmr.search.validators.attribute
   cmr.search.validators.equator-crossing-date
   cmr.search.validators.equator-crossing-longitude
   cmr.search.validators.orbit-number
   cmr.search.validators.temporal
   cmr.search.validators.validation))

(defn- make-deleted-granules-query
  "With given params, construct query for deleted granules"
  [params]
  (let [{:keys [parent-collection-id provider revision-date]} params
        filters [{:range {:revision-date {:gte revision-date}}}]
        filters (if (seq provider)
                  (conj filters {:term {:provider-id provider}})
                  filters)
        filters (if (seq parent-collection-id)
                  (conj filters {:term {:parent-collection-id parent-collection-id}})
                  filters)]
    {:query {:bool {:must (esq/match-all)
                    :filter {:bool {:filter filters}}}}
     :_source [:concept-id :revision-date :provider-id
               :granule-ur :parent-collection-id]}))
