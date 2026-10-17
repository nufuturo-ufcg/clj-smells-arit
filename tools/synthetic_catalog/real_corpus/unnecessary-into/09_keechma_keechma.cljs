(ns keechma.util
  (:require [cljs.core.async :refer [chan close! put!]]))

(defn without [list val]
  (into [] (remove (fn [ls-val]
                     (= val ls-val)) list)))
