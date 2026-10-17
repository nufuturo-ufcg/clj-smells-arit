(ns cmr.indexer.data.concepts.collection.distributed-format-util
  "Defines a utility to parse out multiple distribution format values as a
   string into vector components."
  (:require
   [clojure.string :as string]))

(defn- remove-empty-strings
  "Remove empty strings from the passed in vector"
  [vector]
  (into []
    (remove #(string/blank? %) vector)))
