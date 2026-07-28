(ns namespace-load-side-effects)

(require '[clojure.set :as set])

(defn load-sym [sym]
  (requiring-resolve sym))
