(ns leiningen.new.io.github.kit-clj
  (:require
    [leiningen.new.templates
     :refer [name-to-path sanitize-ns project-name ->files]]
    [leiningen.new.io.github.kit-clj.options.base :as base]
    [leiningen.new.io.github.kit-clj.options.helpers :as helpers]
    [leiningen.new.io.github.kit-clj.options.sql :as sql]
    [io.github.kit-clj.deps-template.helpers :refer [generate-cookie-secret]]
    [clojure.edn :as edn]
    [clojure.java.io :as io]
    [clojure.set :as set]
    [clojure.walk :as walk]))

(def versions (-> (io/resource "io/github/kit_clj/kit/versions.edn")
                  (slurp)
                  (edn/read-string)
                  (walk/keywordize-keys)))
