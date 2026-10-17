(ns host-contract-parameters
  (:import (java.sql PreparedStatement)))

(deftype Bag [^objects items]
  java.util.Collection
  (^objects toArray [this ^objects dest]
    (reduce (fn [idx item]
              (aset dest idx item)
              (inc idx))
            0 (seq items))))

(defn fill! [^objects dest x]
  (aset dest 0 x))

(extend-protocol Param
  clojure.lang.IPersistentVector
  (set-parameter [v ^PreparedStatement stmt ^long idx]
    (let [conn (.getConnection stmt)]
      (.setObject stmt idx (.createArrayOf conn "text" (to-array v))))))

(defn open-conn [ds]
  (let [conn (.getConnection ds)]
    (.prepareStatement conn "select 1")))
