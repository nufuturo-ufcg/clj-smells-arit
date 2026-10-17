(ns testcases.core-async-example.misc
  (:require
   [clojure.core.async :refer [go]]
   [clojure.spec.alpha :as spec]
   [clojure.test :refer [are assert-expr deftest do-report is join-fixtures testing use-fixtures]])
  (:import
   (java.io PrintWriter)))

(defn sample6 [stream ^PrintWriter pw]
  (go
    (with-out-str
      (-> pw (.print "foo"))
      (spit stream 42)
      1)))
