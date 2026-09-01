(ns resolution-semantics-cljc
  #?(:clj (:require [clojure.string :as str])
     :cljs (:require [clojure.string :as str])))

(defn trim-value [value]
  (str/trim value))
