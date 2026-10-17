(ns cljs.analyzer
  #?(:clj  (:refer-clojure :exclude [ensure macroexpand-1])
     :cljs (:refer-clojure :exclude [ensure js-reserved macroexpand-1 ns-interns]))
  #?(:cljs (:require-macros [cljs.analyzer.macros
                             :refer [allowing-redef disallowing-ns* disallowing-recur
                                     no-warn with-warning-handlers wrapping-errors]]
             [cljs.env.macros :refer [ensure]]))
  #?(:clj  (:require [cljs.analyzer.impl :as impl]
                     [cljs.analyzer.impl.namespaces :as nses]
                     [cljs.analyzer.passes.and-or :as and-or]
                     [cljs.analyzer.passes.lite :as lite]
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
                     [cljs.analyzer.passes.lite :as lite]
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

#?(:clj
   (def transit-read-opts
     (try
       (require '[cljs.vendor.cognitect.transit])
       (when-some [ns (find-ns 'cljs.vendor.cognitect.transit)]
         (let [read-handler     @(ns-resolve ns 'read-handler)
               read-handler-map @(ns-resolve ns 'read-handler-map)]
           {:handlers
             (read-handler-map
               {"cljs/js"    (read-handler (fn [v] (JSValue. v)))
                "cljs/regex" (read-handler (fn [v] (Pattern/compile v)))})}))
       (catch Throwable t
         nil))))
