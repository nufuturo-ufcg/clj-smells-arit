(ns cljs.cli
  (:require [clojure.java.io :as io]
            [clojure.string :as string]
            [clojure.edn :as edn]
            [cljs.util :as util]
            [cljs.env :as env]
            [cljs.analyzer :as ana]
            [cljs.analyzer.api :as ana-api]
            [cljs.compiler.api :as comp]
            [cljs.build.api :as build]
            [cljs.closure :as closure]
            [cljs.repl :as repl])
  (:import [java.io File StringReader FileWriter]
           [java.text BreakIterator]
           [java.util Locale]))

(defn watch-proc [cenv path opts]
  (let [log-file (io/file (util/output-directory opts) "watch.log")]
    (util/mkdirs log-file)
    (#'repl/err-out (println "Watch compilation log available at:" (str log-file)))
    (let [log-out (FileWriter. log-file)]
      (binding [*err* log-out
                *out* log-out]
        (build/watch path (dissoc opts :watch) cenv)))))
