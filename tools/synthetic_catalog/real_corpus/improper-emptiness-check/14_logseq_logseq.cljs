(ns frontend.handler.export.opml
  "export blocks/pages as opml"
  (:require ["/frontend/utils" :as utils]
            [clojure.string :as string]
            [clojure.zip :as z]
            [frontend.db :as db]
            [frontend.db.conn :as conn]
            [frontend.format.mldoc :as mldoc]
            [frontend.handler.export.common :as common]
            [frontend.handler.export.zip-helper :refer [get-level goto-last
                                                        goto-level]]
            [frontend.util :as util]
            [hiccups.runtime :as h]
            [frontend.handler.export.common-impl :as common-impl :refer
             [*state* raw-text simple-asts->string space]]
            [frontend.handler.export.util :refer-macros [concatv mapcatv removev]]))

(defn- inline-macro
  [{:keys [name arguments]}]
  (->
   (if (= name "cloze")
     (string/join "," arguments)
     (let [l (cond-> ["{{" name]
               (pos? (count arguments)) (conj "(" (string/join "," arguments) ")")
               true (conj "}}"))]
       (string/join l)))
   raw-text
   vector))
