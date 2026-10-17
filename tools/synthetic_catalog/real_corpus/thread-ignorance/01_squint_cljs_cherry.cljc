(ns cherry.vendor.cljs.analyzer
  #?(:clj  (:refer-clojure :exclude [ensure macroexpand-1])
     :cljs (:refer-clojure :exclude [ensure js-reserved macroexpand-1 ns-interns]))
  #?(:cljs (:require-macros [cljs.analyzer.macros
                             :refer [allowing-redef disallowing-ns* disallowing-recur
                                     no-warn with-warning-handlers wrapping-errors]]
             [cljs.env.macros :refer [ensure]]))
  #?(:clj  (:require [cljs.analyzer.impl :as impl]
                     [cljs.analyzer.impl.namespaces :as nses]
                     [cljs.analyzer.passes.and-or :as and-or]
                     [cljs.env :as env :refer [ensure]]
                     [cljs.externs :as externs]
                     [cljs.js-deps :as deps]
                     [cljs.tagged-literals :as tags]
                     [cljs.util :as util :refer [ns->relpath topo-sort]]
                     [clojure.edn :as edn]
                     [clojure.java.io :as io]
                     [clojure.set :as set]
                     [clojure.string :as string]
                     [cljs.vendor.clojure.tools.reader :as reader]
                     [cljs.vendor.clojure.tools.reader.reader-types :as readers])
     :cljs (:require [cljs.analyzer.impl :as impl]
                     [cljs.analyzer.impl.namespaces :as nses]
                     [cljs.analyzer.passes.and-or :as and-or]
                     [cljs.env :as env]
                     [cljs.reader :as edn]
                     [cljs.tagged-literals :as tags]
                     [cljs.tools.reader :as reader]
                     [cljs.tools.reader.reader-types :as readers]
                     [clojure.set :as set]
                     [clojure.string :as string]
                     [goog.string :as gstring]))
  #?(:clj (:import [cljs.tagged_literals JSValue]
                   [clojure.lang Namespace Var LazySeq ArityException]
                   [java.io File Reader PushbackReader]
                   [java.lang Throwable]
                   [java.net URL]
                   [java.util.regex Pattern])))

(def ^:dynamic *cljs-file* nil)

(defn get-line [x env]
  (or (-> x meta :line) (:line env)))

(defn get-col [x env]
  (or (-> x meta :column) (:column env)))

(defn source-info
  ([env]
   (when (:line env)
     (source-info nil env)))
  ([name env]
   (cond-> {:file (if (= (-> env :ns :name) 'cljs.core)
                    "cljs/core.cljs"
                    *cljs-file*)
            :line (get-line name env)
            :column (get-col name env)}
     (:root-source-info env)
     (merge (select-keys env [:root-source-info])))))

(defn message [env s]
  (str s
    (if (:line env)
      (str " at line " (:line env) " " *cljs-file*)
      (when *cljs-file*
        (str " in file " *cljs-file*)))))

(defn error
  ([env msg]
   (error env msg nil))
  ([env msg cause]
   (ex-info (message env msg)
     (assoc (source-info env) :tag :cljs/analysis-error)
     cause)))

(defn parse-ns-error-msg [spec msg]
  (str msg "; offending spec: " (pr-str spec)))

(defn parse-import-spec [env deps spec]
  (when-not (or (and (sequential? spec)
                     (every? symbol? spec))
                (and (symbol? spec) (nil? (namespace spec))))
    (throw (error env (parse-ns-error-msg spec "Only lib.ns.Ctor or [lib.ns Ctor*] spec supported in :import"))))
  (let [import-map (cond
                     (sequential? spec)
                     (->> (rest spec)
                       (map #(vector % (symbol (str (first spec) "." %))))
                       (into {}))

                     (not (== -1 (.indexOf (str spec) ".")))
                     {(symbol (last (string/split (str spec) #"\."))) spec}

                     :else {})]
    (doseq [[_ spec] import-map]
      (swap! deps conj spec))
    {:import  import-map
     :require import-map}))
