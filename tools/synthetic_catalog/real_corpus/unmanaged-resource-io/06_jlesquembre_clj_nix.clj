(ns cljnix.utils
  (:require
    [clojure.java.shell :as sh]
    [clojure.string :as string]
    [clojure.java.io :as io]
    [clojure.edn :as edn]
    [clojure.data.json :as json]
    [clojure.tools.deps.util.maven :as mvn]
    [clojure.tools.gitlibs.config :as gitlibs-config]
    [clojure.tools.gitlibs.impl :as gli]
    [clojure.tools.deps :as deps]
    [babashka.fs :as fs]
    [babashka.http-client :as http]
    [version-clj.core :as version]
    [clojure.zip :as zip]
    [borkdude.rewrite-edn :as r]
    [clojure.tools.deps.util.io :refer [printerrln]]
    [clojure.tools.deps.extensions.pom :refer [read-model-file]])
  (:import
    [org.apache.maven.model.io.xpp3 MavenXpp3Reader]
    [org.apache.maven.model Model Repository Dependency]
    [java.net URI]))

(defn- pom
  ^Model [pom-path]
  (try
    (when (= "pom" (fs/extension pom-path))
      (let [f (io/input-stream (str pom-path))]
        (.read (MavenXpp3Reader.) f)))
    (catch Exception _
      (printerrln "Error parsing [skiping]" pom-path))))
