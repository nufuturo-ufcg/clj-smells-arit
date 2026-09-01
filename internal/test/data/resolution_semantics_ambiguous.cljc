(ns resolution-semantics-ambiguous
  #?(:clj (:require [clojure.string :as str])
     :cljs (:require [clojure.core :as str])))

(defn trim-value [value]
  (str/trim value))
