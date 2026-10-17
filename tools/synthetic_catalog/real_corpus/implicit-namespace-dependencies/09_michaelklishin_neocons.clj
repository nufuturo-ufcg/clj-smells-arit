(ns clojurewerkz.neocons.rest.cypher
  (:refer-clojure :exclude [empty?])
  (:require [cheshire.core             :as json]
            [clojurewerkz.neocons.rest :as rest]
            [clojurewerkz.support.http.statuses :refer :all]
            [clojurewerkz.neocons.rest.helpers  :refer :all]
            [clojurewerkz.neocons.rest.records  :refer :all])
  (:import  [java.net URI URL]
            clojurewerkz.neocons.rest.Connection
            clojurewerkz.neocons.rest.records.CypherQueryResponse))

(defn empty?
  "Returns true if provided Cypher response is empty (has no data columns), false otherwise.
   Empty responses can be returned by queries but also commonly mutating Cypher (with Neo4J Server 1.8+)"
  [^CypherQueryResponse response]
  (clojure.core/empty? (:data response)))
