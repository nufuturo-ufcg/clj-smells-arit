(ns datahike.json
  "JSON related translations."
  (:require [clojure.edn :as edn]
            [clojure.walk :as walk]
            [datahike.store :refer [store-identity]]
            [datahike.readers :as readers]
            [datahike.connector]
            [datahike.datom :as dd]
            [datahike.db.interface :as dbi]
            [datahike.schema :as s]
            [jsonista.core :as j]
            [jsonista.tagged :as jt])
  #?(:clj
     (:import [datahike.datom Datom]
              [datahike.impl.entity Entity]
              [com.fasterxml.jackson.core JsonGenerator]
              [datahike.db HistoricalDB AsOfDB SinceDB])))

(defn- xf-val [f v]
  (if (vector? v) (map f v) (f v)))

(defn- xf-ref-val [v valtype-attrs-map db]
  (if (vector? v)
    (walk/prewalk #(handle-id-or-av-pair % valtype-attrs-map db) v)
    v))

(defn keywordize-string [s]
  (if (string? s) (keyword s) s))

(defn ident-for [^datahike.db.DB db a]
  (if (and (number? a) (some? db)) (dbi/ident-for db a :error-on-missing) a))

(defn cond-xf-val
  [a-ident v {:keys [ref-attrs long-attrs keyword-attrs symbol-attrs] :as valtype-attrs-map} db]
  (cond
    (contains? ref-attrs a-ident) (xf-ref-val v valtype-attrs-map db)
    (contains? long-attrs a-ident) (xf-val long v)
    (contains? keyword-attrs a-ident) (xf-val keyword v)
    (contains? symbol-attrs a-ident) (xf-val symbol v)
    :else v))

(defn handle-id-or-av-pair
  ([v valtype-attrs-map]
   (handle-id-or-av-pair v valtype-attrs-map nil))
  ([v valtype-attrs-map db]
   (if (and (vector? v) (= (count v) 2))
     (let [a (keywordize-string (first v))]
       [a (cond-xf-val (ident-for db a) (nth v 1) valtype-attrs-map db)])
     v)))

(defn- xf-tx-data-vec [tx-vec valtype-attrs-map db]
  (let [op (first tx-vec)
        [e a v] (rest tx-vec)
        a (keywordize-string a)]
    (vec (filter some? (list (keyword op)
                             (handle-id-or-av-pair e valtype-attrs-map db)
                             a
                             (cond-xf-val (ident-for db a) v valtype-attrs-map db))))))
