(ns pointer.core
  "Functions to help in finding the lines you care about."
  (:require [clojure.string :as string]
            [clojure.set :refer [superset?]]
            [clojure.zip :as zip]))

(defn- this-filename []
  (.getFileName ^StackTraceElement (second (.getStackTrace (Throwable.)))))

(defn current-file-name []
  ;; clojure.test sometimes runs with *file* bound to #"NO_SOURCE.*".
  ;; This corrects that by looking up the stack. Note that it
  ;; produces a reasonable result for the repl, because the stack
  ;; frame it finds has NO_SOURCE_FILE as its "filename".
  (if-not (or (nil? *file*)
              (re-find #"NO_SOURCE" *file*))
    (basename *file*)
    (this-filename)))

(defmacro line-number-known
  "Guess the filename of a file position, but use the given line number."
  [number]
  `[(current-file-name) ~number])

(defn- basename [string]
  (last (string/split string #"/")))
