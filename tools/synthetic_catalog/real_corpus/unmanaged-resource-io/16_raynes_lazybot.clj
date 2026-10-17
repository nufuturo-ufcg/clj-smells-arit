(ns lazybot.plugins.lastfm
  (:require [me.raynes.least :as least]
            [clojure.java.io :refer [file]]
            [clojure.string :refer [join]]
            [clojure.tools.reader.edn :as edn]
            [lazybot.info :refer [*lazybot-dir*]]
            [lazybot.registry :refer [send-message defplugin]])
  (:import java.io.FileWriter))

(def association-file
  (file *lazybot-dir* "lastfmassociations.clj"))

(defn write-associations [assocs]
  (binding [*out* (FileWriter. association-file)]
    (prn assocs)))
