(ns com.fulcrologic.fulcro.react.hooks
  "React hooks wrappers and helpers. The wrappers are simple API simplifications that help when using hooks from
   Clojurescript, but this namespace also includes utilities for using Fulcro's data/network management from raw React
   via hooks.

   See `use-root`, `use-component`, and `use-uism`."
  #?(:cljs
     (:require-macros [com.fulcrologic.fulcro.react.hooks :refer [use-effect use-lifecycle]]))
  ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
  ;; WARNING TO MAINTAINERS: DO NOT REFERENCE DOM IN HERE. This has to work with native.
  ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
  (:require
    #?(:clj [clojure.set :as set])
    #?(:clj [com.fulcrologic.fulcro.react.hooks-context :as hooks-ctx])
    [com.fulcrologic.fulcro.algorithms.tempid :as tempid]
    #?(:cljs ["react" :as react])
    [com.fulcrologic.fulcro.components :as comp]
    [com.fulcrologic.fulcro.raw.components :as rc]
    [com.fulcrologic.fulcro.raw.application :as rapp]
    [com.fulcrologic.fulcro.algorithms.denormalize :as fdn]
    [com.fulcrologic.fulcro.algorithms.merge :as merge]
    [com.fulcrologic.fulcro.ui-state-machines :as uism]
    [com.fulcrologic.fulcro.algorithms.normalized-state :as fns]
    [edn-query-language.core :as eql]
    [com.fulcrologic.fulcro.algorithms.core :as core]
    [taoensso.timbre :as log]
    [com.fulcrologic.fulcro.application :as app])
  ;; babashka can't resolve the cljs compiler's JSValue class; :bb omits the import (it's only used
  ;; to emit #js in cljs macros). :bb must precede :clj since babashka also matches :clj.
  #?@(:bb [] :clj [(:import (cljs.tagged_literals JSValue))]))

(defn useEffect
  "A CLJC wrapper around js/React.useEffect that does NO conversion of
  dependencies. You probably want the macro use-effect instead.

  React docs: https://reactjs.org/docs/hooks-reference.html#useeffect"
  ([f]
   #?(:cljs (react/useEffect f)
      :clj  (useEffect f nil)))
  ([f deps]
   #?(:cljs (react/useEffect f deps)
      :clj  (when (hooks-enabled?)
              (let [path        hooks-ctx/*current-path*
                    idx         (advance-hook-index!)
                    prev-effect (get-effect path idx)
                    ;; deps can be nil (always run), [] (run once), or [a b c] (run when changed)
                    ;; Convert js array to vector if needed
                    deps-vec    (when deps (vec deps))
                    should-run? (or (nil? prev-effect)      ; First mount
                                  (nil? deps-vec)           ; No deps = always run
                                  (not (deps-equal? (:deps prev-effect) deps-vec)))]
                (when should-run?
                  ;; Schedule effect to run after render completes
                  (swap! *pending-effects* conj
                    {:path         path
                     :idx          idx
                     :create       f
                     :deps         deps-vec
                     :prev-destroy (:destroy prev-effect)}))
                ;; Always update deps even if we don't run, so next render can compare
                (when (and prev-effect (not should-run?))
                  (set-effect! path idx (assoc prev-effect :deps deps-vec))))))))

#?(:bb
   (defmacro use-effect
     "babashka variant of `use-effect` (no cljs js-array conversion)."
     ([f] `(useEffect ~f))
     ([f dependencies] `(useEffect ~f ~dependencies)))
   :clj
   (defmacro use-effect
     "A simple macro wrapper around React/useEffect that does compile-time conversion of `dependencies` to a js-array
     for convenience without affecting performance.

      React docs: https://reactjs.org/docs/hooks-reference.html#useeffect"
     ([f]
      `(useEffect ~f))
     ([f dependencies]
      (if (core/compiling-cljs?)
        (let [deps (cond
                     (nil? dependencies) nil
                     (instance? JSValue dependencies) dependencies
                     :else (JSValue. dependencies))]
          `(useEffect ~f ~deps))
        `(useEffect ~f ~dependencies)))))
