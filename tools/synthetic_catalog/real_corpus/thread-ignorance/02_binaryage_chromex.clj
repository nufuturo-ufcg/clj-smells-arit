(ns api-gen.generator
  (:require [clojure.walk :refer [postwalk]]
            [clojure.string :as string]
            [clojure.pprint :refer :all]
            [camel-snake-kebab.core :refer :all]
            [clostache.parser :refer :all]
            [cuerdas.core :as cuerdas]
            [api-gen.word-wrap :refer [wrap]]
            [clj-time.core :as time]
            [clj-time.format :as time-format]
            [api-gen.helpers :refer :all]))

(defn number-of-trailing-optional-arguments-except-callbacks [params]
  (let [is-callback? (fn [param] (= (:type param) :callback))]
    (count (take-while :optional? (reverse (remove is-callback? params))))))
