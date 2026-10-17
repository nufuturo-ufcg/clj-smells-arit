(ns emmy.value
  "The home of most of the protocol-based extensible generic operations offered by
  Emmy. The bulk of the others live in [[emmy.generic]].

  See [the `Generics`
  cljdocs](https://cljdoc.org/d/org.mentat/emmy/CURRENT/doc/basics/generics)
  for a detailed discussion of how to use and extend the generic operations
  defined in [[emmy.generic]] and [[emmy.value]]."
  (:refer-clojure :exclude [zero? number? = compare])
  (:require #?@(:cljs [[emmy.bigfraction :as bf]
                       [emmy.util :as u]
                       [goog.array :as garray]
                       [goog.object :as gobject]
                       [goog.math.Long]
                       [goog.math.Integer]])
            [clojure.core :as core])
  #?(:clj
     (:import
      (clojure.lang BigInt Sequential))))

(defprotocol INumericTower)
