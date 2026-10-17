(ns clojure.core.typed.runtime-infer
  (:refer-clojure :exclude [any? #?(:cljs -val)])
  #?(:cljs
     (:require-macros [clojure.core.typed.runtime-infer
                       :refer [debug-output
                               debug-output-when
                               when-fuel]]))
  (:require [#?(:clj clojure.pprint :cljs cljs.pprint) :as pp]
            [#?(:clj clojure.core :cljs cljs.core) :as core]
            [clojure.core.typed.annotator.rep :refer [-val key-path map-vals-path
                                                      infer-result infer-results
                                                      -class type? -any fn-dom-path
                                                      fn-rng-path -nothing
                                                      seq-entry transient-vector-entry
                                                      index-path vec-entry-path
                                                      set-entry
                                                      make-HMap
                                                      var-path
                                                      alias?
                                                      -alias
                                                      HMap?
                                                      val?
                                                      HVec?
                                                      union?
                                                      nothing?]]
            [clojure.core.typed.annotator.track :refer [track-var'
                                                        track-def-init
                                                        gen-track-config
                                                        track-local-fn
                                                        *track-depth*
                                                        *track-count*
                                                        *root-results*
                                                        local-fn-symbol?
                                                        loop-var-symbol?]]
            [clojure.core.typed.annotator.join :refer [make-Union
                                                       flatten-unions
                                                       flatten-union
                                                       merge-HMaps
                                                       join-HMaps
                                                       HMap-join-strategies
                                                       join*
                                                       join
                                                       join-IFn1]]
            [clojure.core.typed.annotator.env :refer [results-atom
                                                      initial-results
                                                      infer-results?]]
            [clojure.core.typed.annotator.frontend.spec
             :refer [def-spec or-spec unparse-spec unparse-spec'
                     alias-matches-key-for-spec-keys?
                     alias->spec-kw
                     *higher-order-fspec*
                     spec-cat
                     unq-spec-nstr
                     register-unique-alias-for-spec-keys
                     envs-to-specs
                     ]]
            [clojure.core.typed.annotator.debug-macros
             :refer [debug-flat
                     debug-when
                     time-if-slow
                     debug-squash
                     debug]]
            [clojure.set :as set]
            [clojure.string :as str]
            [clojure.core.typed.util-vars :as vs]
            [clojure.core.typed.current-impl :as impl]
            [clojure.math.combinatorics :as comb]
            [clojure.core.typed.contract-utils :as con]
            [clojure.core.typed.annotator.util :refer [unparse-type spec-ns core-specs-ns
                                                       qualify-typed-symbol
                                                       qualify-spec-symbol
                                                       qualify-core-symbol
                                                       *ann-for-ns*
                                                       current-ns
                                                       namespace-alias-in
                                                       camel-case
                                                       *debug*
                                                       update-env
                                                       update-type-env-in-ns
                                                       update-type-env
                                                       update-alias-env
                                                       list*-force
                                                       fv
                                                       resolve-alias
                                                       get-env
                                                       type-env
                                                       alias-env
                                                       *envs*
                                                       unp
                                                       *spec*
                                                       intersection-or-empty
                                                       HMap-common-req-keys
                                                       HMap-req-keyset
                                                       map-key-set
                                                       HMap-likely-tag-key
                                                       kw-val?
                                                       kw-vals?
                                                       *preserve-unknown*
                                                       kw->sym
                                                       *used-aliases*
                                                       *multispecs-needed*
                                                       fully-resolve-alias
                                                       *forbidden-aliases*
                                                       find-top-level-var
                                                       arglists-for-top-level-var
                                                       alternative-arglists
                                                       separate-fixed-from-rest-arglists
                                                       uniquify
                                                       gen-unique-alias-name
                                                       #?@(:clj [macro-symbol?])
                                                       imported-symbol?
                                                       register-alias
                                                       top-level-self-reference?
                                                       ]]
            [clojure.core.typed.annotator.pprint :refer [pprint pprint-str-no-line
                                                         unp-str]]
            [clojure.core.typed.annotator.parse :refer [parse-type prs]]
            [clojure.walk :as walk]
            #?@(:clj [[clojure.tools.namespace.parse :as nprs]
                      [clojure.tools.reader.reader-types :as rdrt]
                      [clojure.java.io :as io]
                      [clojure.core.typed.ast-utils :as ast]
                      [clojure.core.typed.analyzer.jvm.passes.emit-form :as emit-form]
                      [clojure.core.typed.analyzer.jvm :as jana2]
                      [clojure.core.typed.annotator.insert :as insert
                       :refer [replace-generated-annotations]]
                      [clojure.core.typed.coerce-utils :as coerce]])))

(defn walk-type-children [v f]
  {:pre [(type? v)]
   :post [(type? %)]}
  (case (:op v)
    (:val :alias :unknown :Top) v
    :HMap (let [up (fn [m]
                     (reduce-kv
                       (fn [m k v]
                         (assoc m k (f v)))
                       {}
                       m))]
            (-> v
                (update :clojure.core.typed.annotator.rep/HMap-req up)
                (update :clojure.core.typed.annotator.rep/HMap-opt up)))
    (:class :unresolved-class)
           (update v :args (fn [m]
                             (reduce-kv
                               (fn [m k v]
                                 (assoc m k (f v)))
                               []
                               m)))
    :HVec (update v :vec (fn [m]
                           (into []
                                 (map f)
                                 m)))
    :union (apply join* (into #{}
                              (map f)
                              (:types v)))
    :IFn (update v :arities
                 (fn [as]
                   (mapv f as)))
    :IFn1 (-> v
              (update :dom
                      (fn [ds]
                        (mapv f ds)))
              (update :rest (fn [a] (some-> a f)))
              (update :rng f))))

(defn walk
  "Walk the ast applying `pre` when entering the nodes, and `post` when exiting.
   Both functions must return a valid node since the returned value will replace
   the node in the AST which was given as input to the function.
   Short-circuits on reduced."
  ([ast pre post]
     (unreduced
      ((fn walk [ast pre post]
         (let [walk #(walk % pre post)]
           (if (reduced? ast)
             ast
             (let [ret (walk-type-children (pre ast) walk)]
               (if (reduced? ret)
                 ret
                 (post ret))))))
       ast pre post))))
