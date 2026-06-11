(ns blocking-inside-go
  (:require [clojure.core.async :refer [go <!! >!!]]))

(go
  (let [x (<!! ch)]
    x))

(go
  (>!! ch 42))

(defn safe-take []
  (<!! ch))
