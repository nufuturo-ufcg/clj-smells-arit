(ns misuse-of-channel-closing-semantics-invalid
  (:require [clojure.core.async :as a]))

(defn malformed-operations [channel]
  (a/put! channel)
  (a/<! channel :done)
  (= :done (a/<! channel :extra)))
