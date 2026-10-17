(ns game.core.diffs
  (:require
   [cond-plus.core :refer [cond+]]
   [differ.core :as differ]
   [game.core.board :refer [installable-servers]]
   [game.core.card :refer :all]
   [game.core.card-defs :refer [card-def]]
   [game.core.cost-fns :refer [card-ability-cost]]
   [game.core.engine :refer [can-trigger?]]
   [game.core.effects :refer [any-effects is-disabled-reg? get-effects]]
   [game.core.installing :refer [corp-can-pay-and-install?
                                 runner-can-pay-and-install?]]
   [game.core.payment :refer [can-pay? ->c]]
   [game.core.play-instants :refer [can-play-instant?]]
   [game.core.winning :refer [agenda-points-required-to-win]]
   [game.utils :refer [dissoc-in]]
   [jinteki.utils :refer [select-non-nil-keys]]
   [medley.core :refer [update-existing]]))

(defn- fake-log-diff [old new]
  (let [old (:log old)
        new (:log new)
        changes (take-last (- (count new) (count old)) new)]
    (if (seq changes)
      [{:log (mapcat (fn [change] [:+ change]) changes)} {}]
      [{} {}])))

(defn- get-message-diff [old-state new-state side]
  (let [old-messages {:log (into [] (keep #(or (side %) (:public %)) (:log old-state)))}
        new-messages {:log (into [] (keep #(or (side %) (:public %)) (:log @new-state)))}]
    (fake-log-diff old-messages new-messages)))
