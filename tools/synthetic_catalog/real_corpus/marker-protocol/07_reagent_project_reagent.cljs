(ns reagent.ratom
  (:refer-clojure :exclude [atom])
  (:require-macros [reagent.ratom])
  (:require [reagent.impl.util :as util]
            [reagent.debug :refer-macros [warn dev?]]
            [reagent.impl.batching :as batch]
            [clojure.set :as s]
            [goog.object :as obj]))

(defprotocol IReactiveAtom)
