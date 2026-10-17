(ns clojure.core.match
  (:refer-clojure :exclude [compile])
  (:use [clojure.core.match.protocols])
  (:require [clojure.set :as set])
  (:import [java.io Writer]
           [clojure.core.match.protocols IExistentialPattern IPseudoPattern]))

(defn val-at*
  ([m k] (val-at m k ::not-found))
  ([m k not-found] (val-at m k not-found)))

(defn val-at-expr [& args]
  (if *clojurescript*
    `(get ~@args)
    ;;If not ClojureScript, defer to val-at*
    `(if (instance? clojure.lang.ILookup ~(first args))
       (get ~@args)
       (val-at* ~@args))))

(deftype PatternRow [ps action bindings]
  Object
  (equals [_ other]
    (and (instance? PatternRow other)
         (= ps (:ps other))
         (= action (:action other))
         (= bindings (:bindings other))))
  
  IVecMod
  (drop-nth [_ n]
    (PatternRow. (drop-nth ps n) action bindings))

  (prepend [_ x]
    (PatternRow. (into [x] ps) action bindings))

  (swap [_ n]
    (PatternRow. (swap ps n) action bindings))

  clojure.lang.Associative
  (assoc [this k v]
    (PatternRow. (assoc ps k v) action bindings))

  clojure.lang.Indexed
  (nth [_ i]
    (nth ps i))
  (nth [_ i x]
    (nth ps i x))

  clojure.lang.ISeq
  (first [_] (first ps))

  (next [_]
    (if-let [nps (next ps)]
      (PatternRow. nps action bindings)
      (PatternRow. [] action bindings)))

  (more [_]
    (if (empty? ps)
      nil
      (let [nps (rest ps)]
        (PatternRow. nps action bindings))))

  (seq [this]
    (seq ps))

  (count [_]
    (count ps))

  clojure.lang.ILookup
  (valAt [this k]
    (.valAt this k nil))
  (valAt [this k not-found]
    (case k
      :ps ps
      :action action
      :bindings bindings
      not-found))

  clojure.lang.IFn
  (invoke [_ n]
    (nth ps n))

  clojure.lang.IPersistentCollection
  (cons [_ x]
    (PatternRow. (conj ps x) action bindings))
  (equiv [this other]
    (.equals this other)))

(defn pattern-row
  ([ps action] 
    (pattern-row ps action []))
  ([ps action bindings]
    (let [ps (if (vector? ps) ps (into [] ps))]
      (PatternRow. ps action bindings))))

(defn drop-nth-bind [prow n ocr]
  (let [ps        (:ps prow)
        p         (ps n)
        action    (:action prow)
        bind-expr (leaf-bind-expr ocr)
        as        (-> p meta :as)
        bindings  (or (:bindings prow) [])
        bindings  (if as
                    (conj bindings [as bind-expr])
                    bindings)
        bindings  (if (named-wildcard-pattern? p)
                    (conj bindings [(:sym p) bind-expr])
                    bindings)]
    (pattern-row (drop-nth ps n) action bindings)))

(defmulti leaf-bind-expr (fn [ocr] (-> ocr meta :occurrence-type)))

(defmethod leaf-bind-expr :seq
  [ocr] (-> ocr meta :bind-expr))

(defmethod leaf-bind-expr ::vector
  [ocr] (-> ocr meta :bind-expr))

(defmethod leaf-bind-expr :map
  [ocr] (let [m (meta ocr)]
          (val-at-expr (:map-sym m) (:key m))))

(defmethod leaf-bind-expr :default
  [ocr] ocr)

(deftype WildcardPattern [sym named _meta]
  Object
  (equals [_ other]
    (and (instance? WildcardPattern other)
         (if named
           (= sym (:sym other))
           (not (:named other)))))

  clojure.lang.IObj
  (withMeta [_ new-meta]
    (WildcardPattern. sym named new-meta))
  (meta [_]
    _meta)

  clojure.lang.ILookup
  (valAt [this k]
    (.valAt this k nil))
  (valAt [this k not-found]
    (case k
      :sym sym
      :named named
      not-found)))

(defn named-wildcard-pattern? [x]
  (and (instance? WildcardPattern x) (:named x)))
