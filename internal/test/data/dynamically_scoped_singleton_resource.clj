(ns dynamically-scoped-singleton-resource
  (:require [clojure.java.jdbc :as jdbc]
            [example.db :as custom]))

(def ^:dynamic *current-conn* nil)

(defn canonical-jdbc-call []
  (jdbc/query *current-conn* ["SELECT 1"]))

(defn external-alias-call [*current-conn*]
  (custom/query *current-conn* ["SELECT 1"]))

(defn ordinary-argument [connection]
  (jdbc/query connection ["SELECT 1"]))

(defn unresolved-call [*current-conn*]
  (jdbc-unknown/query *current-conn* ["SELECT 1"]))
