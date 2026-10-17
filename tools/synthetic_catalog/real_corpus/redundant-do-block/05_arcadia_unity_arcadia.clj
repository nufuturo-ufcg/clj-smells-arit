(ns arcadia.internal.callbacks
  (:require [arcadia.internal.map-utils :as mu])
  (:import [UnityEngine Debug]
           [System.Collections Queue]
           [Arcadia Callbacks Callbacks+IntervalData]))

(defn safe-dequeue-all [^Queue queue]
  (locking queue
    (when (> (.Count queue) 0)
      (let [objs (.ToArray queue)]
        (.Clear queue)
        objs))))

(defn run-callbacks [work-queue run-repeating-callbacks-fn]
  (let [ar (safe-dequeue-all work-queue)]
    (loop [i (int 0)]
      (when (< i (count ar))
        (do ((aget ar i))
            (recur (inc i))))))
  (run-repeating-callbacks-fn))
