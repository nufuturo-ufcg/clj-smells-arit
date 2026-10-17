(ns ^:typed.clojure ^:no-doc typed.cljc.checker.cs-gen
  (:refer-clojure :exclude [#?(:clj requiring-resolve)])
  (:require [clojure.core.typed :as t :refer [letfn>]]
            [clojure.core.typed.coerce-utils :as coerce]
            [clojure.core.typed.contract-utils :as con]
            [clojure.core.typed.current-impl :as impl]
            [clojure.core.typed.errors :as err]
            [clojure.core.typed.util-vars :as vs]
            [clojure.set :as set]
            #?(:clj [io.github.frenchy64.fully-satisfies.requiring-resolve :refer [requiring-resolve]])
            [typed.clj.checker.parse-unparse :as prs]
            [typed.clj.checker.subtype :as sub] ; use subtype? utility defined in this namespace
            [typed.cljc.checker.check :as check]
            [typed.cljc.checker.cs-rep :as cr]
            [typed.cljc.checker.proposition-ops :as fo]
            [typed.cljc.checker.proposition-rep :as fr]
            [typed.cljc.checker.fold-rep :as f]
            [typed.cljc.checker.free-ops :as free-ops]
            [typed.cljc.checker.frees :as frees]
            [typed.cljc.checker.indirect-ops :as ind]
            [typed.cljc.checker.object-rep :as or]
            [typed.cljc.checker.promote-demote :as prmt]
            [typed.cljc.checker.subst :as subst]
            [typed.cljc.checker.type-ctors :as c]
            [typed.cljc.checker.type-rep :as r]
            [typed.cljc.checker.utils :as u])
  (:import (typed.cljc.checker.cs_rep c cset dcon dmap cset-entry)
           (typed.cljc.checker.type_rep ArrayCLJS AssocType B Bounds CLJSInteger CountRange DataType DissocType DottedPretype F FnIntersection
                                        Function GetType HSequential HSet HeterogeneousMap Instance Intersection JSBoolean JSNominal JSNull
                                        JSNumber JSObj JSObject JSString JSSymbol JSUndefined KwArgs KwArgsArray KwArgsSeq MatchType MergeType
                                        Mu Name NotType Poly PrimitiveArray Protocol RClass Regex Result Satisfies Scope SymbolicClosure TApp
                                        TCError TCResult Top TopFunction TopHSequential TopKwArgsSeq TypeFn TypeOf Unchecked Union Value Wildcard)))

(def inferrable-symbolic-closure-expected-type?
  (comp (some-fn r/FnIntersection? r/Poly? r/PolyDots?)
        c/fully-resolve-type))

(defn fail! [s t]
  (throw u/cs-gen-exn))

(defn prep-symbolic-closure-expected-type [substitution-without-symb dom-t opts]
  {:pre [(cr/substitution-c? substitution-without-symb)
         (r/Type? #_inferrable-symbolic-closure-expected-type? dom-t)]
   :post [(r/Type? #_inferrable-symbolic-closure-expected-type? %)]}
  ;(prn :prep-symbolic-closure-expected-type dom-t)
  (let [var-hash (frees/fv+idx-variances dom-t opts)
        subst (reduce-kv (fn [subst k v]
                           (let [variance (var-hash k)]
                             (cond-> subst
                               variance (assoc k (case variance
                                                   :covariant (cond
                                                                (cr/t-subst? v) (assoc v :type r/-wild)
                                                                :else v)
                                                   (:invariant :contravariant) v)))))
                         {} substitution-without-symb)]
    (subst/subst-all subst dom-t opts)))

(defn prep-symbolic-closure-expected-type2 [subst t opts]
  {:pre [(cr/substitution-c? subst)
         (inferrable-symbolic-closure-expected-type? t opts)]
   :post [(r/Type? %)]}
  ;(prn "prep-symbolic-closure-expected-type2" t)
  ;(prn "subst" subst)
  (let [t (c/fully-resolve-type t opts)]
    (cond
      (r/FnIntersection? t) (-> t
                                (update :types (fn [types]
                                                 (mapv (fn [{:keys [rng] :as f}]
                                                         {:pre [(r/Function? f)]
                                                          :post (r/Function? %)}
                                                         (let [rng-var-hash (frees/fv+idx-variances rng opts)
                                                               ;_ (prn "rng-var-hash" rng-var-hash rng)
                                                               rng-subst (reduce-kv (fn [subst k v]
                                                                                      (let [variance (rng-var-hash k)]
                                                                                        ;(prn "variance" k variance)
                                                                                        (cond-> subst
                                                                                          variance (assoc k (case variance
                                                                                                              :covariant (cond
                                                                                                                           (cr/t-subst? v) (assoc v :type r/-wild)
                                                                                                                           :else v)
                                                                                                              (:invariant :contravariant) v)))))
                                                                                    {} subst)]
                                                           (-> f
                                                               (assoc :rng (r/make-Result r/-any))
                                                               (as-> t (subst/subst-all subst t opts))
                                                               (assoc :rng (subst/subst-all rng-subst rng opts)))))
                                                       types))))
      (r/Poly? t) (let [names (c/Poly-fresh-symbols* t)
                        bbnds (c/Poly-bbnds* names t opts)
                        opts (free-ops/with-bounded-frees opts (zipmap (map r/make-F names) bbnds))
                        body (c/Poly-body* names t opts)]
                   (c/Poly* names bbnds (prep-symbolic-closure-expected-type2 subst body opts) opts))
      (r/PolyDots? t) (let [names (c/PolyDots-fresh-symbols* t)
                            bbnds (c/PolyDots-bbnds* names t opts)
                            opts  (free-ops/with-bounded-frees opts (zipmap (map r/make-F names) bbnds))
                            body  (c/PolyDots-body* names t opts)]
                        (c/PolyDots* names bbnds (prep-symbolic-closure-expected-type2 subst body opts) opts))
      :else (do ;(prn "unsupported type in prep-symbolic-closure-expected-type: " t)
                (fail! nil nil)))))
