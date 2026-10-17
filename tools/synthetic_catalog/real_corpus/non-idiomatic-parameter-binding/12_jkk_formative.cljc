(ns formative.util
  (:require [clojure.string :as string]
            #?@(:clj [[clj-time.core :as ct]
                      [clj-time.coerce :as cc]
                      [clj-time.format :as cf]]))
  (:import #?@(:clj [org.joda.time.DateTime
                     org.joda.time.LocalDate
                     org.joda.time.LocalTime])))

(defn seqify-value [s & [split-re]]
  (cond
    (or (sequential? s) (nil? s)) s
    (string? s) (if (string/blank? s)
                  []
                  (string/split s (or split-re #"\s*,\s*")))
    (nil? s) s
    :else [s]))
