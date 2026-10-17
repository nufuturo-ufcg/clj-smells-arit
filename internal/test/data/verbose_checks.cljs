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

;; pos?/neg?/inc/dec are defined as these exact forms in ClojureScript.
(defn cljs-definitional [x]
  [(> x 0) (< 0 x) (+ x 1) (- x 1)])

;; JavaScript + concatenates in operand order: (+ 1 "a") is "1a".
(defn cljs-leading-one [x]
  (+ 1 x))

;; NaN makes the non-strict rewrite differ.
(defn cljs-non-strict [x]
  (>= x 0))
