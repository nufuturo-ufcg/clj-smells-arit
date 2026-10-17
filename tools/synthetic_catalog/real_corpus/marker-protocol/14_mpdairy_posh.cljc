(ns posh.lib.ratom
  "Ported to .cljc from reagent.ratom by alexandergunnarson."
           (:refer-clojure :exclude [atom run!])
           (:require        [clojure.set           :as s])
  #?(:cljs (:require-macros [posh.lib.ratom
                              :refer [getm setm! getf setf! add! array-list alength* aset* aget*]]))
  #?(:clj  (:import         [java.util ArrayList]
                            [clojure.lang IDeref IAtom IRef IMeta IHashEq])))

(defprotocol IReactiveAtom)
