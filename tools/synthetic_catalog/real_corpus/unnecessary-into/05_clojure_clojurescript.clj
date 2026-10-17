(ns cljs.externs
  (:require [cljs.util :as util]
            [cljs.js-deps :as js-deps]
            [clojure.java.io :as io]
            [clojure.string :as string])
  (:import [com.google.javascript.jscomp
            CompilerOptions CompilerOptions$Environment SourceFile CompilerInput CommandLineRunner]
           [com.google.javascript.jscomp.parsing Config$JsDocParsing]
           [com.google.javascript.rhino
            Node Token JSTypeExpression JSDocInfo JSDocInfo$Visibility]
           [java.util.logging Level]
           [java.net URL]))

(def ^:dynamic *goog-ns* nil)

(defn ns-match? [ns-segs var-segs]
  (or
    ;; exact match (i.e. ctors)
    (= ns-segs var-segs)
    (and
      (= (inc (count ns-segs)) (count var-segs))
      (= ns-segs (take (count ns-segs) var-segs)))))

(defmulti parsed->defs (fn [_ module-type] module-type))

(defmethod parsed->defs :goog
  ([externs _]
   (let [grouped (group-by #(= 'exports (first %)) externs)
         exports (->> (get grouped true)
                   (map (comp vec rest))
                   (remove empty?)
                   set)
         exported (filter exports (get grouped false))]
     (reduce
       (fn [m xs]
         (let [sym (last xs)]
           (cond-> m
             (seq xs) (assoc sym (merge (meta sym) {:ns *goog-ns* :name sym})))))
       {} exported))))

(defmethod parsed->defs :default
  ([externs _]
   (let [ns-segs (into [] (map symbol (string/split (str *goog-ns*) #"\.")))]
     (reduce
       (fn [m xs]
         ;; ignore definitions from other provided namespaces not under consideration
         (if (ns-match? ns-segs xs)
           (let [sym (last xs)]
             (cond-> m
               (seq xs) (assoc sym (merge (meta sym) {:ns *goog-ns* :name sym}))))
           m))
       {} externs))))
