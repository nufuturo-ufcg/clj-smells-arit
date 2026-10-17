(ns datahike.db.interface
  (:require [replikativ.logging :as log]))

(defrecord SearchContext [historical temporal timepred xform currentdb])

(defn- extend-pred [pred added-pred]
  (if (nil? pred)
    added-pred
    (fn [x] (and (pred x) (added-pred x)))))

(defn context-with-temporal-timepred [^SearchContext c timepred]
  (SearchContext. (.-historical c)
                  true
                  (extend-pred (.-timepred c) timepred)
                  (.-xform c)
                  (.-currentdb c)))
