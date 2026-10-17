(ns metabase.driver.bigquery-cloud-sdk.params
  (:require
   [java-time.api :as t]
   [metabase.util.date-2 :as u.date]
   [metabase.util.log :as log])
  (:import
   (com.google.cloud.bigquery QueryJobConfiguration$Builder QueryParameterValue StandardSQLTypeName)))

(defmulti ^:private ->QueryParameterValue
  {:arglists '(^QueryParameterValue [v])}
  class)

(defmethod ->QueryParameterValue QueryParameterValue [v] v)

(defmethod ->QueryParameterValue :default
  [v]
  (param "STRING" v))

(defmethod ->QueryParameterValue nil
  [_]
  (param "STRING" nil))

(defmethod ->QueryParameterValue String               [v] (param "STRING" v))

(defmethod ->QueryParameterValue Boolean              [v] (param "BOOL" v))

(defmethod ->QueryParameterValue Integer              [v] (param "INT64" v))

(defmethod ->QueryParameterValue Long                 [v] (param "INT64" v))

(defmethod ->QueryParameterValue Short                [v] (param "INT64" v))

(defmethod ->QueryParameterValue Byte                 [v] (param "INT64" v))

(defmethod ->QueryParameterValue clojure.lang.BigInt  [v] (param "INT64" v))

(defmethod ->QueryParameterValue java.math.BigInteger [v] (param "INT64" v))

(defmethod ->QueryParameterValue Float                [v] (param "FLOAT64" v))

(defmethod ->QueryParameterValue Double               [v] (param "FLOAT64" v))

(defmethod ->QueryParameterValue java.math.BigDecimal [^BigDecimal v]
  (if (or (and (pos? (.signum v))
               (pos? (.compareTo v min-bq-numeric-val)))
          (and (neg? (.signum v))
               (neg? (.compareTo v max-bq-numeric-val))))
    (param "BIGNUMERIC" v)
    (param "NUMERIC" v)))

(defmethod ->QueryParameterValue java.time.LocalDate      [t] (param "DATE" (u.date/format t)))

(defmethod ->QueryParameterValue java.time.LocalDateTime  [t] (param "DATETIME" (u.date/format t)))

(defmethod ->QueryParameterValue java.time.LocalTime      [t] (param "TIME" (u.date/format t)))

(defmethod ->QueryParameterValue java.time.OffsetTime     [t] (param "TIME" (->> (t/zone-offset 0)
                                                                                 (t/with-offset-same-instant t)
                                                                                 t/local-time
                                                                                 u.date/format)))

(defmethod ->QueryParameterValue java.time.OffsetDateTime [t] (param "TIMESTAMP" (u.date/format t)))

(defmethod ->QueryParameterValue java.time.ZonedDateTime  [t] (param "TIMESTAMP" (->> (t/zone-id "UTC")
                                                                                      (t/with-zone-same-instant t)
                                                                                      t/offset-date-time
                                                                                      u.date/format)))
