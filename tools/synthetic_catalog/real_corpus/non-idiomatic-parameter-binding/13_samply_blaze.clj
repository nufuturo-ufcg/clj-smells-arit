(ns blaze.fhir-client
  (:refer-clojure :exclude [read spit str update])
  (:require
   [blaze.async.comp :as ac]
   [blaze.async.flow :as flow]
   [blaze.fhir-client.impl :as impl]
   [blaze.util :refer [str]]
   [taoensso.timbre :as log])
  (:import
   [java.util.concurrent Flow$Publisher]))

(defn create
  "Returns a CompletableFuture that will complete with `resource` created."
  {:arglists '([base-uri resource & [opts]])}
  [base-uri {:fhir/keys [type] :as resource} & [opts]]
  (impl/create (str base-uri "/" (name type)) resource opts))
