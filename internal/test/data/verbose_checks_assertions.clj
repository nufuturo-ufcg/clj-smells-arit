(ns verbose-checks-assertions
  (:require [clojure.test :as t]))

(t/deftest canonical-assertions
  ;; clojure.test assertions preserve expected/actual diagnostics.
  (t/is (= nil (external-value)))
  (t/is (= true (external-flag)))
  (t/is (= 0 (external-count))))
