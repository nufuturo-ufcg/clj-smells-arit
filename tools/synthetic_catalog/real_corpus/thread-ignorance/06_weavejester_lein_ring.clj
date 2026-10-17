(ns leiningen.ring.war
  (:require [leiningen.compile :as compile]
            [leiningen.ring.war.manifest
             :refer [make-manifest]]
            [clojure.java.io :as io]
            [clojure.string :as string]
            [leinjacker.deps :as deps])
  (:use [clojure.data.xml :only [sexp-as-element indent-str]]
        leiningen.ring.util)
  (:import [java.util.jar JarEntry
                          JarOutputStream]
           [java.io BufferedOutputStream
                    FileOutputStream
                    ByteArrayInputStream
                    File]))

(defn war-resources-paths [project]
  (filter identity
          (distinct (concat [(:war-resources-path project "war-resources")]
                            (:war-resource-paths project)))))
