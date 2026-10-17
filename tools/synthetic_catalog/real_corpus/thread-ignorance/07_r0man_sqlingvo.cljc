(ns sqlingvo.compiler
  #?(:cljs (:require-macros [sqlingvo.compiler :refer [defarity]]))
  (:require [clojure.core :as core]
            [clojure.string :as str]
            [sqlingvo.expr :as expr]
            [sqlingvo.util :as util :refer [sql-quote sql-quote-fn]]))

(defn to-sql [arg]
  (cond
    (string? arg)
    [arg]
    (sequential? arg)
    arg))

(defn join-sql [separator args]
  (let [args (map to-sql args)]
    (cons (str/join separator (remove str/blank? (map first args)))
          (apply concat (map rest args)))))
