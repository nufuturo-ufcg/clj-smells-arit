(ns ^{:mranderson/inlined true} day8.re-frame-10x.inlined-deps.reagent.v1v2v0.reagent.ratom
  (:refer-clojure :exclude [atom])
  (:require-macros [day8.re-frame-10x.inlined-deps.reagent.v1v2v0.reagent.ratom])
  (:require [day8.re-frame-10x.inlined-deps.reagent.v1v2v0.reagent.impl.util :as util]
            [day8.re-frame-10x.inlined-deps.reagent.v1v2v0.reagent.debug :refer-macros [warn dev?]]
            [day8.re-frame-10x.inlined-deps.reagent.v1v2v0.reagent.impl.batching :as batch]
            [clojure.set :as s]
            [goog.object :as obj]))

(defprotocol IReactiveAtom)
