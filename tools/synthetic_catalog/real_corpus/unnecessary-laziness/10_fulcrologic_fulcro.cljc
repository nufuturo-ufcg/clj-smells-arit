(ns com.fulcrologic.fulcro.ui-state-machines
  "Support for structuring the logic of component interactions as a state machine. Useful when
   the logic of a group of components becomes non-trivial, which is often. Note that it is not
   recommended that you try to structure large parts of your application as a state machine
   (since this implementation is closer to FSM than state charts), but it turns out that
   most applications work very well with groups of components under separate state machines
   as somewhat larger modules of the overall application. See the Developer's Guide for
   more information on working with this namespace."
  #?(:cljs (:require-macros com.fulcrologic.fulcro.ui-state-machines))
  (:refer-clojure :exclude [load])
  (:require
    #?(:cljs [goog.functions :as gf])
    [clojure.set :as set]
    [clojure.spec.alpha :as s]
    [clojure.string :as str]
    [com.fulcrologic.fulcro.algorithms.data-targeting :as targeting]
    [com.fulcrologic.fulcro.algorithms.denormalize :as fdn]
    [com.fulcrologic.fulcro.algorithms.do-not-use :as util :refer [atom?]]
    [com.fulcrologic.fulcro.algorithms.lookup :as ah]
    [com.fulcrologic.fulcro.algorithms.scheduling :as sched]
    [com.fulcrologic.fulcro.algorithms.tempid :as tempid]
    [com.fulcrologic.fulcro.algorithms.tx-processing :as txn]
    [com.fulcrologic.fulcro.data-fetch :as df]
    [com.fulcrologic.fulcro.mutations :as m :refer [defmutation]]
    [com.fulcrologic.fulcro.raw.application :as rapp]
    [com.fulcrologic.fulcro.raw.components :as rc]
    [com.fulcrologic.guardrails.core :refer [>defn => | ? <- >def]]
    [edn-query-language.core :as eql]
    [taoensso.timbre :as log]))

(>defn remove-ident
  "Removes an ident, if it exists, from an alias that points to a list of idents."
  [env ident alias-to-idents]
  [::env eql/ident? ::alias => ::env]
  (let [new-list (fn [old-list]
                   (vec (filter #(not= ident %) old-list)))]
    (update-aliased env alias-to-idents new-list)))
