(ns control.main
  (:use [control.core])
  (:use [clojure.string :only [join]]
        [clojure.tools.cli :only [cli]]
        [clojure.java.io :only [file reader writer]])
  (:gen-class))

(def ^:private special-control-file (atom nil))

(defn- get-control-dir []
  (System/getProperty CONTROL-DIR "."))

(defn- create-control-ns []
  (create-ns (gensym "user-control")))

(defn- load-control-file []
  (try 
    (binding [*ns* (create-control-ns)]
      (refer-clojure)
      (use '[control core commands])
      (load-file
       (or @special-control-file (str (get-control-dir) "/control.clj"))))
    (catch java.io.FileNotFoundException e (error "control file not found."))))

(defn- run-control [ & args]
  (do
    (load-control-file)
    (do-begin args)
    (shutdown-agents)))
