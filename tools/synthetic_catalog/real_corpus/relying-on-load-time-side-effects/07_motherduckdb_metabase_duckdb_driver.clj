(ns metabase.driver.duckdb
  (:require
   [clojure.java.jdbc :as jdbc]
   [clojure.string :as str]
   [java-time.api :as t]
   [medley.core :as m]
   [metabase.driver :as driver]
   [metabase.driver.sql-jdbc.common :as sql-jdbc.common]
   [metabase.driver.sql-jdbc.connection :as sql-jdbc.conn]
   [metabase.driver.sql-jdbc.execute :as sql-jdbc.execute]
   [metabase.driver.sql-jdbc.sync :as sql-jdbc.sync]
   [metabase.driver.sql.query-processor :as sql.qp]
   [metabase.util.honey-sql-2 :as h2x]
   [metabase.util.log :as log])
  (:import
   (java.sql
    Connection
    PreparedStatement
    ResultSet
    ResultSetMetaData
    Statement
    Time
    Types)
   (java.time LocalDate LocalTime OffsetTime)
   (java.time.temporal ChronoField)))

(def premium-features-namespace
  (try
    (require '[metabase.premium-features.core :as premium-features])    ;; For Metabase 0.52 or after
    'metabase.premium-features.core
    (catch Exception _
      (try
        (require '[metabase.public-settings.premium-features :as premium-features])   ;; For Metabase < 0.52
        'metabase.public-settings.premium-features
        (catch Exception e
          (throw (ex-info "Could not load either premium features namespace"
                          {:error e})))))))
