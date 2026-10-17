(ns reifyhealth.specmonstah.core
  (:require
   [better-cond.core :as b]
   [clojure.data :as data]
   [clojure.set :as set]
   [clojure.spec.alpha :as s]
   [clojure.string :as str]
   #?(:bb      [loom.alg-generic :as lgen]
      :default [loom.alg :as la])
   [loom.attr :as lat]
   [loom.derived :as ld]
   [loom.graph :as lg]
   [medley.core :as medley]))

#?(:bb
   nil
   :clj
   (do
     (require '[loom.io :as lio])
     (defn view
       "View with loom"
       [{:keys [data]}]
       (lio/view data))))
