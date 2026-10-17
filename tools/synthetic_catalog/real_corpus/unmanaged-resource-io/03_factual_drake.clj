(ns drake.fs
  (:refer-clojure :exclude [file-seq])
  (:require [fs.core :as fs]
            [drake.plugins :as plugins]
            [hdfs.core :as hdfs]
            [clojure.string :as s :refer [join split]]
            [aws.sdk.s3 :as s3]
            [drake-interface.core :as di]
            [slingshot.slingshot :refer [throw+]]
            [drake.shell :refer [shell]]
            [drake.options :refer [*options*]])
  (:import org.apache.hadoop.conf.Configuration
           (org.apache.hadoop.fs Path FileStatus)
           java.io.File))

(defn- load-props
  "Loads a java style properties file into a struct map."
  [filename]
  (let [file (fs/file filename)]
    (when-not (fs/exists? file)
      (throw+ {:msg (format "unable to locate properties file %s" filename)}))
    (let [io (java.io.FileInputStream. file)
          prop (java.util.Properties.)]
      (.load prop io)
      (into {} prop))))
