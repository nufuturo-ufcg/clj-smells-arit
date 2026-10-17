(ns cmr.sample-data.util
  (:require
   [cheshire.core :as json]
   [clojure.java.io :as io]
   [clojure.string :as string]
   [cmr.sample-data.const :as const])
  (:import
   (java.util.zip ZipFile)))

(defn parse-jar-file
  [dir]
  (-> (io/resource dir)
      (.getPath)
      (string/split #"!")
      first
      (string/split #"file:")
      last))

(defn jar-file?
  [entry]
  (not (.isDirectory entry)))

(defn jar-file-starts-with?
  [jar-file dir]
  (-> jar-file
      (.getName)
      (string/starts-with? dir)))

(defn get-jar-files
  [dir]
  (let [jar-file (parse-jar-file dir)
        zip-file (new ZipFile jar-file)
        entries (enumeration-seq (.entries zip-file))]
    (->> (.entries zip-file)
         enumeration-seq
         (filter jar-file?)
         (filter #(jar-file-starts-with? % dir))
         (map #(io/resource (.getName %))))))
