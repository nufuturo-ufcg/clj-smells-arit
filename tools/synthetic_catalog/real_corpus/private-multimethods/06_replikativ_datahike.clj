(ns datahike.norm.norm
  (:require
   [clojure.java.io :as io]
   [clojure.string :as string]
   [clojure.pprint :as pp]
   [clojure.data :as data]
   [clojure.edn :as edn]
   [clojure.spec.alpha :as s]
   [replikativ.logging :as log]
   [hasch.core :as h]
   [hasch.platform :as hp]
   [datahike.api :as d]
   [datahike.tools :as dt])
  (:import
   [java.io File]
   [java.util.jar JarFile JarEntry]
   [java.net URL]))

(defn- get-jar [resource]
  (-> (.getPath resource)
      (string/split #"!" 2)
      first
      (subs 5)
      JarFile.))

(defmulti ^:private retrieve-file-list
  (fn [file-or-resource] (type file-or-resource)))

(defmethod ^:private retrieve-file-list File [file]
  (if (.exists file)
    (let [migration-files (file-seq file)
          xf (comp
              (filter #(.isFile %))
              (filter #(string/ends-with? (.getPath %) ".edn")))]
      (into [] xf migration-files))
    (log/raise (format "Norms folder %s does not exist." (str file)) {:folder file})))

(defmethod ^:private retrieve-file-list URL [resource]
  (if resource
    (let [abs-path (.getPath resource)
          last-path-segment (-> abs-path (string/split #"/") peek)]
      (if (string/starts-with? abs-path "file:")
        (->> (get-jar resource)
             .entries
             enumeration-seq
             (filter #(and (string/starts-with? (.getName %) last-path-segment)
                           (not (.isDirectory %))
                           (string/ends-with? % ".edn"))))
        (->> (file-seq (io/file abs-path))
             (filter #(not (.isDirectory %))))))
    (log/raise "Resource does not exist." {:resource (str resource)})))

(defmethod ^:private retrieve-file-list :default [arg]
  (log/raise "Can only read a File or a URL (resource)" {:arg arg :type (type arg)}))
