(ns direct-use-of-clojure-lang-rt-alias
  (:require [other.runtime :as RT]))

(defn aliased-rt [coll]
  (RT/count coll))
