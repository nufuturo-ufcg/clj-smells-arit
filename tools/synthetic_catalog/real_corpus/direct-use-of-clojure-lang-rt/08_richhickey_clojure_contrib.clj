(ns
  #^{:author "Laurent Petit (and others)"
     :doc "Functions/macros variants of the ones that can be found in clojure.core 
 (note to other contrib members: feel free to add to this lib)"}
  clojure.contrib.core
  (:use clojure.contrib.def))

(defn new-by-name
  "Constructs a Java object whose class is specified by a String."
  [class-name & args]
  (clojure.lang.Reflector/invokeConstructor
   (clojure.lang.RT/classForName class-name)
   (into-array Object args)))
