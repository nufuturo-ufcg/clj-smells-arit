(ns verbose-checks-foreign-deftest
  (:require [some.dsl :refer :all]))

(deftest counts
  (> (:n {:n 1}) 0))
