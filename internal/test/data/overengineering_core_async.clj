(ns overengineering-core-async
  (:require [clojure.core.async :as a]
            [example.async :as custom]))

(defn one-value [x]
  (let [c (a/chan 1)]
    (a/go (a/>! c (inc x)) (a/close! c))
    c))

(defn actual-stream [xs]
  (let [c (a/chan)]
    (a/go (doseq [x xs] (a/>! c x)))
    c))

(defn two-values [x y]
  (let [c (a/chan 2)]
    (a/put! c x)
    (a/put! c y)
    c))

(defn repeated-value [x]
  (let [c (a/chan 1)]
    (a/go
      (while true (a/>! c x)))
    c))

(defn callback-bridge [x]
  (let [c (a/chan 1)]
    (proxy [java.nio.channels.CompletionHandler] []
      (completed [value attachment] (a/put! c value)))
    c))

(defn external-channel [x]
  (let [c (custom/chan 1)]
    (custom/put! c x)
    c))

(defn local-channel-name [x]
  (let [chan (fn [_] nil)
        c (chan 1)]
    c))

;; A transducer transforms what is put, so the channel is not a direct value.
(defn incremented [v]
  (let [c (a/chan 1 (map inc))]
    (a/put! c v)
    c))
