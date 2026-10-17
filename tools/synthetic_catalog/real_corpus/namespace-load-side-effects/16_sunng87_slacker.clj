(ns slacker.serialization
  (:require [clojure.tools.logging :as logging]
            [clojure.java.io :refer [copy]]
            [slacker.common :refer :all]
            [clojure.edn :as edn])
  (:import [java.nio.charset Charset]
           [io.netty.buffer ByteBuf ByteBufAllocator ByteBufInputStream
            ByteBufOutputStream]))

(defn- resolve-by-name [ns mem]
  @(ns-resolve (symbol ns) (symbol mem)))

(defmulti serialize
  "serialize clojure data structure to bytebuf with
  different types of serialization"
  (fn [f _ & _] f))

(defmulti deserialize
  "deserialize clojure data structure from bytebuf using
  matched serialization function"
  (fn [f _ & _] f))

(defmethod deserialize :clj
  [_ ^ByteBuf data]
  (let [s (.toString data ^Charset (Charset/forName "UTF-8"))]
    (edn/read-string s)))

(defmethod serialize :clj
  [_ data]
  (let [ednstr (pr-str data)
        bytes (.getBytes ^String ednstr "UTF-8")
        bytes-length (alength bytes)
        buffer (.buffer ByteBufAllocator/DEFAULT bytes-length)]
    (.writeBytes buffer bytes)
    buffer))

(try
  (require '[taoensso.nippy])
  (try
    (require '[slacker.serialization.nippy])
    (catch Throwable _
      (logging/info "Nippy version below 2.7.1. Disable stacktrace transfer support")))

  (let [thaw (resolve-by-name "taoensso.nippy" "thaw-from-in!")
        freeze (resolve-by-name "taoensso.nippy" "freeze-to-out!")]

    (defmethod deserialize :nippy
      [_ ^ByteBuf data]
      (let [bin (ByteBufInputStream. data)]
        (thaw bin)))

    (defmethod serialize :nippy
      [_ data]
      (let [buffer (.buffer ByteBufAllocator/DEFAULT)
            bos (ByteBufOutputStream. buffer)]
        (freeze bos data)
        buffer)))

  (catch Throwable _
    (logging/info "Disable nippy support.")))
