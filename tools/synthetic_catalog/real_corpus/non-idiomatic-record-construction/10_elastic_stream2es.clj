(ns stream2es.stream.generator
  (:refer-clojure :exclude [name type])
  (:require [cheshire.core :as json]
            [clojure.java.io :as jio]
            [clojure.string :refer [split]]
            [stream2es.es :as es]
            [stream2es.http :as http]
            [stream2es.log :as log]
            [stream2es.util.string :refer [rand-str]]
            [stream2es.stream :refer [new specs Stream
                                      Streamable CommandLine
                                      StreamStorage]]))

(defprotocol FieldSpec
  (parse [spec]))

(defprotocol Field
  (make [field opts]))

(defrecord IntField [name size]
  Field
  (make [field opts]
    (map->IntField
     {:type :int
      :name name
      :size size
      :value (rand-int size)})))

(defrecord StringField [name size]
  Field
  (make [field opts]
    (let [v (rand-str (:size field) (::dictionary opts))]
      (map->StringField
       (assoc field :value v )))))

(defrecord DoubleField [name size]
  Field
  (make [field opts]
    (let [v (Double/parseDouble
             (format "%d.%d" (rand-int size) (rand-int 1000000)))]
      (map->DoubleField
       {:type :double
        :name name
        :size size
        :value v}))))

(extend-protocol FieldSpec
  String
  (parse [spec]
    (when (pos? (count (.trim spec)))
      (let [[name type size] (split spec #":")
            size (try (Integer/parseInt size) (catch Exception _ 0))]
        (case type
          "int" (IntField. name size)
          "integer" (IntField. name size)
          "str" (StringField. name size)
          "string" (StringField. name size)
          "dbl" (DoubleField. name size)
          "double" (DoubleField. name size)
          (StringField. name size))))))
