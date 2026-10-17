(ns clojure.test-clojure.numbers
  (:use clojure.test
        [clojure.test.generative :exclude (is)]
        clojure.template)
  (:require [clojure.data.generators :as gen]
            [clojure.test-helper :as helper]))

(defn all-pairs-hash-consistent-with-= [vals]
  (doseq [val1 vals]
    (doseq [val2 vals]
      (when (= val1 val2)
        (is (= (hash val1) (hash val2))
            (str "Test that (hash " val1 ") (" (class val1) ") "
                 " = (hash " val2 ") (" (class val2) ")"))))))
