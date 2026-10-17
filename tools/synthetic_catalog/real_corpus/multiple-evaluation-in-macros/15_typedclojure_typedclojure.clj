(ns ^:typed.clojure ^:no-doc typed.cljc.checker.check.recur-utils
  (:require [typed.clojure :as t]
            [typed.cljc.checker.utils :as u]
            [typed.cljc.checker.type-rep :as r])
  (:import (typed.cljc.checker.type_rep DottedPretype)))

(defmacro ^:private set-validator-doc! [var val-fn]
  `(set-validator! ~var (fn [a#] (assert (~val-fn a#)
                                         (str "Invalid reference state: " ~var
                                              " with value: "
                                              (pr-str a#)))
                          true)))
