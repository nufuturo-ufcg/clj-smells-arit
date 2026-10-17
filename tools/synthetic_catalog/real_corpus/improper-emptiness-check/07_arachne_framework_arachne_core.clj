(ns arachne.core.util
  (:require [clojure.java.io :as io]
            [clojure.spec.alpha :as s]
            [clojure.string :as str]
            [arachne.core.util.specs]
            [arachne.error :as e :refer [deferror error]])
  (:import [java.io FileNotFoundException]
           [java.lang.reflect Method])
  (:refer-clojure :exclude [alias]))

(defn arity
  "Given a function, return its arity (or :many if the function is variadic)"
  [f]
  (let [methods (.getDeclaredMethods (class f))
        invokes (filter #(= "invoke" (.getName ^Method %)) methods)
        do-invokes (filter #(= "doInvoke" (.getName ^Method %)) methods)]
    (cond
      (not (empty? do-invokes)) :many
      (< 1 (count invokes)) :many
      (empty? invokes) (error ::arity-detection-error {:f f})
      :else (count (.getParameterTypes ^Method (first invokes))))))
