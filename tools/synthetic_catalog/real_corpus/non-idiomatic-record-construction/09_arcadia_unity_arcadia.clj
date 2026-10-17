(ns clojure.test-clojure.protocols
  (:use clojure.test clojure.test-clojure.protocols.examples)
  (:require [clojure.test-clojure.protocols.more-examples :as other]
            [clojure.set :as set]
            clojure.test-helper)
  (:import [clojure.test_clojure.protocols.examples ExampleInterface]))

(defrecord DefrecordObjectMethodsWidgetA [a])

(defrecord DefrecordObjectMethodsWidgetB [a])

(deftest defrecord-object-methods-test
  (testing "= depends on fields and type"
    (is (true? (= (DefrecordObjectMethodsWidgetA. 1) (DefrecordObjectMethodsWidgetA. 1))))
    (is (false? (= (DefrecordObjectMethodsWidgetA. 1) (DefrecordObjectMethodsWidgetA. 2))))
    (is (false? (= (DefrecordObjectMethodsWidgetA. 1) (DefrecordObjectMethodsWidgetB. 1))))))
