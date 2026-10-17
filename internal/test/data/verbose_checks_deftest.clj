(ns verbose-checks-deftest
  (:require [clojure.test :refer :all]))

(deftest counts
  (testing "strict comparison inside test bodies is ordinary code"
    (is (true? (> (:n {:n 1}) 0)))))

(defmacro positive? [x]
  (if (> x 0) :yes :no))
