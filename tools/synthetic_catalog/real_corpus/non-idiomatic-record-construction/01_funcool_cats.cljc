(ns cats.monad.identity
  "The Identity Monad."
  (:refer-clojure :exclude [identity])
  (:require [cats.protocols :as p]
            [cats.context :as ctx]
            [cats.util :as util]))

(defrecord Identity [v]
  p/Contextual
  (-get-context [_] context)

  p/Extract
  (-extract [_] v)

  p/Printable
  (-repr [_]
    (str "#<Identity " (pr-str v) ">"))

  #?@(:cljs [cljs.core/IDeref
             (-deref [_] v)]
      :clj  [clojure.lang.IDeref
             (deref [_] v)]))

(defn identity
  "The Identity type constructor."
  [v]
  (Identity. v))
