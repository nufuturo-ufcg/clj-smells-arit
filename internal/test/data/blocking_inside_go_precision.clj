(ns blocking-inside-go-precision
  (:require [clojure.core.async :as a]
            [example.async :as custom]))

(defn local-blocking-name [channel]
  (let [<!! (fn [_] nil)]
    (a/go (<!! channel))))

(defn external-blocking-name [channel]
  (custom/go (custom/<!! channel)))

(defn canonical-blocking-operation [channel]
  (a/go (a/<!! channel)))
