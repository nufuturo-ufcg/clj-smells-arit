(ns qanal.kafka
  (:import (kafka.javaapi.consumer SimpleConsumer))
  (:require [qanal.utils :refer [result-or-exception bytes->string from-json safe-short]]
            [clj-kafka.zk :refer (brokers committed-offset set-offset!) :as zk]
            [clj-kafka.core :refer (ToClojure)]
            [clj-kafka.consumer.simple :refer (consumer topic-meta-data messages topic-offset)]
            [schema.core :as s]
            [samsara.trackit :refer [track-time]]
            [taoensso.timbre :as log])
  (:require [clojure.core.async :as async :refer [<!! >!! timeout chan alt!!]])
  (:require [clj-kafka.consumer.simple :as zs]))

(defmacro safe-exec [message & body]
  `(try
     ~@body
     (catch Exception x#
       (log/warn x# "ERROR: " x# "on:" ~message )
       nil)))

(defmacro forever-do [name sleep & body]
  `(let [_sleep#   (or ~sleep 1000)
         _name#    ~name]
     (future
       (while true
         (safe-exec (str "" _name# ", will retry in " _sleep# "ms")
           ~@body)
         (when (> _sleep# 0)
           (safe-exec (str "" _name# " interrupted while sleeping...")
             (Thread/sleep _sleep#)))))))
