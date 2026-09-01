(ns verbose-checks-cljs
  (:require [cljs.test :as test]))

(defn boolean-check [value]
  (= value true))

;; The numeric domain of an external value is unknown.
(defn numeric-check [value]
  (= value 0))

;; Assertion comparisons are part of the test diagnostic contract.
(defn asserted-numeric-check [value]
  (test/is (= value 0)))

(defn boolean-if [value]
  (if value true false))
