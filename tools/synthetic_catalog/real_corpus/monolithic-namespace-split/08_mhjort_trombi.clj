(ns trombi.simulation-util
  (:require [clojure.java.io :as io]
            [clojure.string :as str])
  (:import [java.io File]
           [java.io StringWriter PrintWriter]
           [java.time Duration LocalDateTime]
           [java.time.format DateTimeFormatter]
           [trombi.simulation_runners FixedRequestNumberRunner DurationRunner]))

(defn symbol-namespace [^clojure.lang.Symbol simulation]
  (str "/"
       (clojure.string/join "/"
                            (-> simulation
                                (str)
                                (clojure.string/replace #"\." "/")
                                (clojure.string/replace #"-" "_")
                                (clojure.string/split #"/")
                                (drop-last)))))

(defn load-namespace [^clojure.lang.Symbol simulation]
  (let [loadable-ns (symbol-namespace simulation)]
    (when (not (= "/" loadable-ns)) ;No need to load if namespace is current ns
      (load loadable-ns))))
