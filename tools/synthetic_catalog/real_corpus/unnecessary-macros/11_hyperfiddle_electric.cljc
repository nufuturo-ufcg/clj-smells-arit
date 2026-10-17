(ns hyperfiddle.electric3
  (:refer-clojure :exclude [fn defn apply letfn for declare #?(:cljs Range)])
  (:require [hyperfiddle.electric.impl.missionary-util :as mu]
            [hyperfiddle.electric.impl.lang3 :as lang]
            [hyperfiddle.electric.impl.runtime3 :as r]
            #?(:clj [hyperfiddle.electric.impl.entrypoint]) ; TODO rename server-entrypoint
            #?@(:cljs [hyperfiddle.electric-client3], :node nil) ; auto-load electric-client in browser builds. Before users had to remember to require it for side effect at the app entrypoint.
            [hyperfiddle.electric-dom3-events :as dom-events]
            [hyperfiddle.incseq :as i]
            [hyperfiddle.electric.impl.mount-point :as mp]
            [clojure.core :as cc]
            [clojure.string :as str]
            [dustingetz.data]
            [dustingetz.measure :as cm]
            [hyperfiddle.rcf :as rcf :refer [tests]]
            #?(:clj [dustingetz.triple-store :as ts])
            [missionary.core :as m]
            [dustingetz.missionary-contrib :as mx]
            [clojure.math :as math]
            [dustingetz.debug :as dbg]
            [dustingetz.check :as ca])
  (:import [missionary Cancelled])
  #?(:cljs (:require-macros hyperfiddle.electric3)))

(defmacro ctor [expr] `(::lang/ctor ~expr))
