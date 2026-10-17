(ns cljss.collect
  (:require [cljss.utils :refer [build-css]]))

(def env* (atom {:id  0
                 :cls nil}))

(defn reset-env! [v]
  (reset! env* (merge {:id 0 :cls nil} v)))
