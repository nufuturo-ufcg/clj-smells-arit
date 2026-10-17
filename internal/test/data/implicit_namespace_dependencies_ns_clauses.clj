(ns implicit-namespace-dependencies-ns-clauses
  (use clojure.test))

;; Outside ns, (:use m) and (:require m) are keyword lookups, not clauses.
(defn lookup [m]
  [(:use m) (:require m)])
