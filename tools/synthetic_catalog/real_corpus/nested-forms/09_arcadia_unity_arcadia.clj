(ns clojure.test-clojure.numbers
  (:use clojure.test
        [clojure.test.generative :exclude (is)]
        clojure.template)
  (:require [clojure.data.generators :as gen]
            [clojure.test-helper :as helper]))

(deftest BigInteger-conversions
  (doseq [coerce-fn [bigint biginteger]]
    (doseq [v (map coerce-fn [ Int32/MaxValue                                  ;;; Long/MAX_VALUE
                              13178456923875639284562345789M
                              13178456923875639284562345789N
                              Single/MaxValue                                  ;;; Float/MAX_VALUE
                              (- Single/MaxValue)                              ;;; (- Float/MAX_VALUE)
                              Double/MaxValue                                  ;;; Double/MAX_VALUE
                              (- Double/MaxValue)                              ;;; (- Double/MAX_VALUE)
                              (* 2 (bigdec Double/MaxValue)) ])]               ;;;  Double/MAX_VALUE
      (are [x] (true? x)
        (integer? v)
        (number? v)
        (not (decimal? v))
        (not (float? v))))))
