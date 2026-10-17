(ns verbose-checks-assertions-referred
  (:require #?(:clj [clojure.test :refer [deftest is]])
            #?(:cljs [cljs.test :refer [deftest is]])))

(deftest referred-is
  (is (= nil (:missing {})))
  (is (= (:ok {:ok true}) true)))
