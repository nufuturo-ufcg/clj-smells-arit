(ns implicit-namespace-dependencies-reader-conditional
  #?(:clj (:use [clojure.test]))
  #?@(:clj [(:require [clojure.string :refer :all])]))

;; (:use m) outside ns is a keyword lookup.
(defn lookup [m] #?(:clj (:use m) :cljs m))
