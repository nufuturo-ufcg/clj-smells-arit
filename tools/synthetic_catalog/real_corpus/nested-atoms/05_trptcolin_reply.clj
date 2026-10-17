(ns reply.eval-modes.standalone.concurrency
  (:require [clojure.main]))

(def actions (atom []))

(defn act-on-form [action-state act form]
  (swap! action-state assoc :thread (Thread/currentThread))
  (let [result (act form)]
    (swap! action-state assoc :completed true)
    result))

(defn act-in-future [act]
  (let [action-state (atom nil)]
    (swap! actions conj action-state)
    (fn [form]
      (try
        (reset! action-state {})
        @(future (act-on-form action-state act form))
        (catch Throwable e
          (throw e))))))
