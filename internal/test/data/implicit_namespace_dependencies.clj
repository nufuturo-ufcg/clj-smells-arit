(ns implicit-namespace-deps-example
  (:use [clojure.set])
  (:use clojure.walk)
  (:require [clojure.string :refer :all]
            [clojure.data :as data]
            [clojure.zip :refer [zipper]]))

;; Standalone use call - should be detected
(use 'clojure.pprint)

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Proper require with alias
(defn proper-alias-usage []
  (data/diff {:a 1} {:b 2}))

;; Proper require with explicit refer
(defn proper-refer-usage [t]
  (zipper sequential? seq identity t))
