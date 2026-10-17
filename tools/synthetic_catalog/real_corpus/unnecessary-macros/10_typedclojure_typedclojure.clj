(ns ^:typed.clojure ^:no-doc typed.cljc.checker.check.recur-utils
  (:require [typed.clojure :as t]
            [typed.cljc.checker.utils :as u]
            [typed.cljc.checker.type-rep :as r])
  (:import (typed.cljc.checker.type_rep DottedPretype)))

(defmacro with-recur-target [opts tgt]
  `(assoc ~opts ::recur-target ~tgt))
