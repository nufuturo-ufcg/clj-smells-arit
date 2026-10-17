(ns clojure.core.typed.checker.fold-default
  (:require [clojure.core.typed.checker.fold-rep :refer [add-default-fold-case]]
            [clojure.core.typed.checker.type-rep :as r]
            [clojure.core.typed.checker.type-ctors :as c]
            [clojure.core.typed.checker.filter-rep]
            [clojure.core.typed.checker.filter-ops :as fops]
            [clojure.core.typed.checker.object-rep]
            [clojure.core.typed.checker.free-ops :as free-ops]
            [clojure.core.typed.checker.jvm.assoc-utils :as assoc-u]
            [clojure.core.typed.checker.path-rep])
  (:import (clojure.core.typed.checker.type_rep NotType DifferenceType Intersection Union FnIntersection Bounds
                                        DottedPretype Function RClass JSNominal App TApp
                                        PrimitiveArray DataType Protocol TypeFn Poly PolyDots
                                        Mu HeterogeneousMap
                                        CountRange Name Value Top Unchecked TopFunction B F Result
                                        TCResult TCError FlowSet Extends
                                        JSNumber CLJSInteger JSObject JSString ArrayCLJS
                                        JSBoolean AssocType GetType KwArgsSeq HSequential HSet
                                        JSUndefined JSNull JSSymbol JSObj TypeOf SymbolicClosure)
           (clojure.core.typed.checker.filter_rep NoFilter TopFilter BotFilter TypeFilter NotTypeFilter
                                          ImpFilter AndFilter OrFilter FilterSet)
           (clojure.core.typed.checker.object_rep NoObject EmptyObject Path)
           (clojure.core.typed.checker.path_rep KeyPE KeysPE ValsPE ClassPE NthPE CountPE KeywordPE)))

(add-default-fold-case Extends
                       (fn [{:keys [extends without] :as ty} _]
                         (c/-extends
                           (doall (map type-rec extends))
                           :without (doall (mapv type-rec without)))))
