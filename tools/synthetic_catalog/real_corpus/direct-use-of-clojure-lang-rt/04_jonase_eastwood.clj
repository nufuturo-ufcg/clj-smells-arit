(ns 
 ^{:author "Stuart Sierra",
   :doc "This namespace is DEPRECATED; most functions have been moved to
  other namespaces"
   :deprecated "0.2.1"
   :added "0.1.0"}
 eastwood.copieddeps.dep9.clojure.tools.namespace
 (:require [clojure.java.io :as io])
 (:import (java.io File FileReader BufferedReader PushbackReader
                   InputStreamReader)
          (java.util.jar JarFile JarEntry)))

(defn- loader-classpath [loader]
  (when (instance? java.net.URLClassLoader loader)
    (map
     #(java.io.File. (.toURI ^java.net.URL %))
     (.getURLs ^java.net.URLClassLoader loader))))

(defn- classpath
  ([classloader]
     (distinct
      (mapcat
       loader-classpath
       (take-while
        identity
        (iterate #(.getParent ^ClassLoader %) classloader)))))
  ([] (classpath (clojure.lang.RT/baseLoader))))
