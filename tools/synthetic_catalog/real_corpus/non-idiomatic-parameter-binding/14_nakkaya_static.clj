(ns static.core
  (:require [clojure.core.memoize :refer [memo-clear!]]
            [clojure.java.browse :as browse]
            [clojure.string :as str]
            [clojure.tools.cli :as cli]
            [clojure.tools.logging :as log]
            [hiccup.core :as hiccup]
            [hiccup.page :refer :all]
            [hiccup.util :refer :all]
            [ring.adapter.jetty :as jetty]
            [ring.util.response :refer :all]
            [static.config :as config]
            [static.io :as io]
            [stringtemplate-clj.core :as string-template]
            [watchtower.core :as watcher])
  (:import (java.io File)
           (java.net URL)
           (java.text SimpleDateFormat)
           (org.apache.commons.io FileUtils FilenameUtils))
  (:gen-class))

(defn site-url [f & [ext]]
  (-> (str f)
      (.replaceAll "\\\\" "/")
      (.replaceAll (io/dir-path :site) "")
      (FilenameUtils/removeExtension)
      (str "."
           (or ext
               (:default-extension (config/config))))))
