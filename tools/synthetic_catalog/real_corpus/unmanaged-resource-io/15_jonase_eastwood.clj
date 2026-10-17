(ns eastwood.analyze-ns
  (:refer-clojure :exclude [macroexpand-1])
  (:require
   [clojure.java.io :as io]
   [clojure.pprint :as pp]
   [clojure.string :as string]
   [clojure.walk :as walk]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.ast :as ast]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.env :as env]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.passes :refer [schedule]]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.passes.trim :refer [trim]]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.passes.warn-earmuff :refer [warn-earmuff]]
   [eastwood.copieddeps.dep10.clojure.tools.reader :as reader]
   [eastwood.copieddeps.dep10.clojure.tools.reader.reader-types :as reader-types]
   [eastwood.copieddeps.dep2.clojure.tools.analyzer.jvm :as jvm]
   [eastwood.copieddeps.dep2.clojure.tools.analyzer.passes.jvm.emit-form :refer [emit-form]]
   [eastwood.copieddeps.dep2.clojure.tools.analyzer.passes.jvm.warn-on-reflection :refer [warn-on-reflection]]
   [eastwood.copieddeps.dep9.clojure.tools.namespace.move :as move]
   [eastwood.passes :as pass]
   [eastwood.util :as util])
  (:import
   (clojure.lang Namespace)))

(def ^:dynamic *forms-read-writer* nil)

(def ^:dynamic *forms-analyzed-writer* nil)

(def ^:dynamic *forms-emitted-writer* nil)

(defn- with-form-writers [opt f]
  (if-not (util/debug? :compare-forms opt)
    (f)
    (binding [*forms-read-writer* (io/writer "forms-read.txt")
              *forms-analyzed-writer* (io/writer "forms-analyzed.txt")
              *forms-emitted-writer* (io/writer "forms-emitted.txt")]
      (f))))
