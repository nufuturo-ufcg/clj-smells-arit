(ns gen-changelog
  (:require [clojure.java.shell :refer [sh]]
            [clojure.string :as str])
  (:import (java.io File))
  (:gen-class))

(def tags
  (->> (sh "git" "tag" "-l")
       :out
       java.io.StringReader.
       java.io.BufferedReader.
       line-seq
       (mapcat (fn [^String s]
                 (when-let [version (some (fn [prefix]
                                            (when (.startsWith s prefix)
                                              (subs s (count prefix))))
                                          ["" "core.typed-pom-"])]
                   [version s])))
       (apply hash-map)))
