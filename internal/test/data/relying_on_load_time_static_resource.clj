(ns relying-on-load-time-static-resource)

(def schema (slurp (clojure.java.io/resource "schema.edn")))

(def threaded-schema
  (-> (clojure.java.io/resource "schema.edn")
      slurp))
