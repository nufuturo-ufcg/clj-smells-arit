(ns browser
  (:require
   ; [cemerick.cljs.test]
   [ajax.protocols :refer [-body]]
   [ajax.core :refer [abort ajax-request
                      url-request-format
                      raw-response-format
                      transit-request-format
                      transit-response-format
                      GET POST]]
   [ajax.edn :refer [edn-request-format
                     edn-response-format]]
   [clojure.string :as s]))
