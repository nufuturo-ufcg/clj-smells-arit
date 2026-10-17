(ns pod.borkdude.clj-kondo
  {:no-doc true}
  (:refer-clojure :exclude [read read-string])
  (:require [bencode.core :as bencode]
            [clj-kondo.core :as clj-kondo]
            [clojure.edn :as edn]
            [clojure.java.io :as io])
  (:import [java.io PushbackInputStream])
  (:gen-class))

(def debug? false)

(defn debug [& args]
  (when debug?
    (binding [*out* (io/writer "/tmp/debug.log" :append true)]
      (apply println args))))
