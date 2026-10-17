(ns cljx.a
        (:use [clojure.pprint :only [pp]]))

(defmacro increment [x] `(inc ~x))
