(ns unmanaged-resource-io-standard-streams
  (:require [clojure.java.io :as io]))
;; Wrapping the process's standard streams opens nothing; closing the wrapper
;; would close stderr or stdout.
(defn to-stderr [s]
  (let [w (io/writer *err*)]
    (.write w s)
    (.flush w)))
(defn to-stdout [s]
  (let [w (java.io.BufferedWriter. *out*)]
    (.write w s)))
(defn shadowed [*err* s]
  (let [w (io/writer *err*)]
    (.write w s)))
(defn to-file [s]
  (let [w (io/writer "out.txt")]
    (.write w s)))
