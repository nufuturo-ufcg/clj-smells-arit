(ns metabase.util.performance
  "Functions and utilities for faster processing. This namespace is compatible with both Clojure and ClojureScript.
  However, some functions are either not only available in CLJS, or offer passthrough non-improved functions."
  (:refer-clojure :exclude [reduce mapv run! some every? concat select-keys update-keys empty? not-empty doseq for
                            first second update-vals get-in #?(:cljs clj->js)])
  #?(:clj (:require
           [net.cgrand.macrovich :as macros])
     :cljs (:require
            [cljs.core :as core]
            [goog.object :as gobject]))
  #?(:clj (:import (clojure.lang ITransientCollection LazilyPersistentVector RT)
                   (java.util ArrayList HashMap Iterator List))))

#?(:clj
   (defn second
     "Like `clojure.core/second`, but uses `RT/nth` directly for allocation-free access."
     [coll]
     (RT/nth coll 1 nil))
   :cljs
   (def second
     "Same as (first (next x))"
     clojure.core/second))
