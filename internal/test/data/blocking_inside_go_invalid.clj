(ns blocking-inside-go-invalid
  (:require [clojure.core.async :as a]))

(defn malformed-blockers [channel]
  (a/<!!)
  (a/>!! channel)
  (Thread/sleep))
