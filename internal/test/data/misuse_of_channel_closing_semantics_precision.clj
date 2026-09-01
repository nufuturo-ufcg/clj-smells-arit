(ns misuse-of-channel-closing-semantics-precision
  (:require [clojure.core.async :as a]
            [example.async :as custom]))

(defn external-put [channel]
  (custom/put! channel :done))

(defn local-put [channel]
  (let [put! (fn [_ _] nil)]
    (put! channel :done)))

(defn external-take [channel]
  (when (= :done (custom/<! channel))
    :done))

(defn local-go [channel]
  (let [go (fn [body] body)]
    (go (when (= :done channel) :done))))

(defn valid-core-operations [channel]
  (a/go
    (a/put! channel :done)))
