(ns cmr.system-int-test.utils.generic-util
  "Utility functions and definitions for use by generic document pipeline tests."
  (:require
   [cheshire.core :as json]
   [clj-http.client :as client]
   [clojure.java.io :as io]
   [clojure.string :as string]
   [cmr.common.mime-types :as mt]
   [cmr.mock-echo.client.echo-util :as echo-util]
   [cmr.system-int-test.system :as sys]
   [cmr.system-int-test.utils.url-helper :as url-helper]
   [cmr.transmit.config :as transmit-config]))

(def citation (-> "schemas/citation/v1.0.0/metadata.json"
                  (io/resource)
                  (slurp)
                  (json/parse-string true)))
