(ns flow-storm.utils
  #?(:cljs (:require [goog.string :as gstr]
                     [clojure.string :as str]
                     [amalloy.ring-buffer :refer [ring-buffer]]
                     [goog.string.format]
                     [goog :as g])
     :clj (:require [clojure.java.io :as io]
                    [amalloy.ring-buffer :refer [ring-buffer]]
                    [clojure.string :as str]))
  (:refer-clojure :exclude [format update-values update-keys])
  #?(:clj (:import [java.io File LineNumberReader InputStreamReader PushbackReader]
                   [clojure.lang RT IEditableCollection PersistentArrayMap PersistentHashMap]
                   [java.util.logging Logger Level])))

(defn grep-coll

  "Search over coll with pred, when it matches returns the
  A elements before and the B elements after"
  
  [coll A B pred]
  (loop [elems-before (ring-buffer A)
         [e & rcoll] coll]
    (when e
      (if (pred e)
        {:before (into [] elems-before)
         :match e
         :after (into [] (take B rcoll))}
        (recur (conj elems-before e)
               rcoll)))))
