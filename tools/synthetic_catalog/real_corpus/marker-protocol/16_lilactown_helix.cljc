(ns helix.hooks
  "Wrapper around react hooks.

  Many functions take a `deps` argument, corresponding to their React
  equivalent.  This is an argument which can either be a a vector of
  deps or a special keyword:

  vector of deps  Use specified deps explicitly.
  :always         Re-run hook on every render, equivalent to passing no deps
                  vector to the hook.
  :once           Equivalent to using [] as the deps.
  :auto-deps      Infer the dependencies automatically from the code by finding
                  local vars.  Not available for the function form of a hook."
  #?(:clj (:require [helix.impl.analyzer :as hana]
                    [helix.impl.hooks :as-alias impl.hooks])
     :cljs (:require
            [helix.impl.hooks :as impl.hooks]
            ["react" :as react]
            [goog.object :as gobj]))
  #?(:cljs (:require-macros [helix.hooks])))

(defprotocol IStateUpdater
  "Protocol that marks a type as callable when passed to a use-state setter.")
