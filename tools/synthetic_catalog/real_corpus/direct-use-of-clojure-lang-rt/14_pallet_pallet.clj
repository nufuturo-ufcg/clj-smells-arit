(ns pallet.template
  "Template file writing"
  (:require
   [clojure.tools.logging :as logging]
   [pallet.actions :refer [remote-file]]
   [pallet.core.session :refer [group-name os-family packager]]
   [pallet.strint :as strint]
   [pallet.utils :as utils]
   [pallet.utils :refer [apply-map]]))

(defn ^java.net.URL get-resource
  "Loads a resource. Returns a URI."
  [path]
  (-> (clojure.lang.RT/baseLoader) (.getResource path)))
