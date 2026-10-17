(ns selmer.filters
  "To create a filter use the function add-filter! which takes a name and a fn.
The first argument to the fn is always the value obtained from the context
map. The rest of the arguments are optional and are always strings."
  (:require
    [clojure.string :as s]
    [selmer.util :as u])
  (:import
    java.util.Locale
    [java.time Instant
               LocalTime
               LocalDate
               LocalDateTime
               ZonedDateTime
               ZoneId]
    [java.time.format DateTimeFormatter FormatStyle]
    java.text.NumberFormat))

(def generate-json
  "JSON generation function. Resolved in the (arbitrary) order of cheshire, clojure.data.json, jsonista. Falls back on function that throws at runtime. It is important for GraalVM native-image that we resolve this at compile time (top level)."
  (try (require 'cheshire.core)
       @(resolve 'cheshire.core/generate-string)
       (catch Exception _
         (try (require 'clojure.data.json)
              @(resolve 'clojure.data.json/write-str)
              (catch Exception _
                (try (require 'jsonista.core)
                     @(resolve 'jsonista.core/write-value-as-string)
                     (catch Exception e
                       (fn [_]
                         (throw (ex-info "Supported JSON libraries: cheshire, clojure.data.json or
                                          jsonista. Provide one of these on the classpath to enable
                                          the json filter." {} e))))))))))
