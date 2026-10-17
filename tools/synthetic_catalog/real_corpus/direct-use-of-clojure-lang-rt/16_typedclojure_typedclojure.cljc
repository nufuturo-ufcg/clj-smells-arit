(ns ^:typed.clojure typed.clj.analyzer.utils
  #?(:clj (:refer-clojure :exclude [delay]))
  (:require [typed.cljc.analyzer.utils :as u]
            [typed.cljc.analyzer :as ana2]
            [clojure.set :as set]
            [clojure.reflect :as reflect]
            [clojure.string :as s]
            #?@(:bb []
                :clj [[clojure.core.memoize :refer [lru]]])
            #?(:cljr [clojure.clr.io]
               :default [clojure.java.io :as io])
            #?(:clj [io.github.frenchy64.fully-satisfies.safe-locals-clearing :refer [delay]]))
  (:import (clojure.lang RT Symbol Var)
           #?@(:bb []
               :clj [org.objectweb.asm.Type])))

(defn resolve-sym
  "Resolves the value mapped by the given sym in the global env
  If sym is shadowed by a local in env, returns nil."
  [sym {:keys [ns locals] :as env}]
  (when (symbol? sym)
    (ns-resolve ns locals sym)))

(defmulti ^#?(:cljr Type :default Class) maybe-class
  "Takes a Symbol, String or Class and tries to resolve to a matching Class"
  class)

(defn array-class
  ([element-type] (array-class 1 element-type))
  ([n element-type]
   (RT/classForName
    #?(:bb (throw (ex-info (str "TODO bb: " `array-class) {}))
       :cljr (apply str (-> element-type
                            maybe-class
                            .FullName
                            (.Replace \/ \.))
                    (repeat n "[]"))
       :default (str (apply str (repeat n "["))
                     (-> element-type
                         maybe-class
                         Type/getType
                         .getDescriptor
                         (.replace \/ \.)))))))

(defn maybe-class-from-string [^String s]
  (or (when-let [maybe-class (and (neg? (#?(:cljr .IndexOf  :default .indexOf) s "."))
                                  (not= \[ (first s))
                                  (resolve-sym (symbol s) {:ns (ns-name *ns*)}))]
        (when (class? maybe-class) maybe-class))
      (try (RT/classForName s)
           (catch #?(:cljr Exception :default ClassNotFoundException) _))))

(defmethod maybe-class :default [_] nil)

(defmethod maybe-class #?(:cljr Type :default Class) [c] c)

(defmethod maybe-class String [s]
  (maybe-class (symbol s)))

(defn maybe-array-class-sym [x]
  (when-some [c (namespace x)]
    (let [sname (name x)]
      (when (and (= (count sname) 1)
                 #?(:cljr (Char/IsDigit (char (first sname)))
                    :default (Character/isDigit (char (first sname)))))
        (when-let [c (or (specials c)
                         (maybe-class-from-string c))]
          (array-class #?(:cljr (Int32/Parse sname)
                          :default (Integer/parseInt sname)) c))))))

(defmethod maybe-class Symbol [sym]
  (if (namespace sym)
    (maybe-array-class-sym sym)
    (let [sname (name sym)
          snamec (count sname)]
      (if-let [base-type (and (#?(:cljr .EndsWith :default .endsWith) sname "<>")
                              (maybe-class (subs sname 0 (- snamec 2))))]
        ;; TODO: we're leaking into the syntax
        (array-class base-type)
        (if-let [ret (or (specials sname)
                         (special-arrays sname))]
          ret
          (maybe-class-from-string sname))))))

(defn maybe-class-literal [x]
  (cond
   (class? x) x
   (symbol? x) (or (maybe-array-class-sym x)
                   (and (not (namespace x))
                        (maybe-class-from-string (name x)))
                   #?(:cljr
                      ;; Try to resolve CLR System.* types
                      (when (and (not (namespace x))
                                 (s/starts-with? (name x) "System."))
                        (try (RT/classForName (name x))
                             (catch Exception _)))))
   (string? x) (maybe-class-from-string x)))
