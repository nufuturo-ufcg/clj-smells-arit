(ns cmr.search.site.util
  "Search site support functions."
  (:require
   [cheshire.core :as json]
   [clj-http.client :as client]
   [clojure.java.io :as io]
   [cmr.common.log :refer [warn]]
   [cmr.transmit.config :as transmit])
  (:import (java.net ConnectException)))

(defn- get-search-reference-file
  "Locate a reference file whose location is well-known and stable (won't
  change) in the search-app codebase. This is intended to be used to calculate
  the absolute file-system path to the `search-app` directory.

  Note, seleection of function depends on ns being in `require`, so it can't
  be something that causes a cyclic dependency."
  []
  (-> (meta #'cmr.search.site.util/get-search-reference-file)
      :file
      (io/resource)
      (.getFile)))
