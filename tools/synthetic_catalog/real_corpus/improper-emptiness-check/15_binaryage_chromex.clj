(ns api-gen.helpers
  (:require [clojure.walk :refer [postwalk]]
            [clojure.string :as string]
            [clojure.pprint :refer :all]
            [camel-snake-kebab.core :refer :all]
            [clostache.parser :refer :all]
            [cuerdas.core :as cuerdas]
            [api-gen.word-wrap :refer [wrap]]
            [cljfmt.core :as cljfmt]
            [clojure.java.io :as io])
  (:import [java.net URLEncoder]))

(defn has-any? [coll]
  (not (empty? coll)))
