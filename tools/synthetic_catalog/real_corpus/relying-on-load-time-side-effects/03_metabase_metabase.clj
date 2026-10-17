(ns metabase.config.core
  (:require
   ^{:clj-kondo/ignore [:discouraged-namespace]}
   [clojure.java.io :as io]
   [clojure.string :as str]
   [environ.core :as env]
   [net.cgrand.macrovich :as macros])
  (:import
   (clojure.lang Keyword)))

(def ^{:doc "Indicates whether Enterprise Edition extensions are available" :added "0.39.0"} ee-available?
  (try
    #_{:clj-kondo/ignore [:metabase/modules]}
    (require 'metabase-enterprise.core.dummy-namespace)
    true
    (catch Throwable _
      false)))
