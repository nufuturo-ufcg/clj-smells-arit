(ns game.cards.events
  (:require
   [clojure.set :as set]
   [clojure.string :as str]
   [game.core.access :refer [access-card breach-server get-only-card-to-access
                             num-cards-to-access]]
   [game.core.agendas :refer [update-all-agenda-points]]
   [game.core.bad-publicity :refer [gain-bad-publicity]]
   [game.core.board :refer [all-active-installed all-installed server->zone]]
   [game.core.card :refer [agenda? asset? card-index condition-counter? corp?
                           event? facedown? faceup? get-card get-counters
                           get-nested-host get-title get-zone hardware? has-subtype? ice? in-discard? in-hand?
                           installed? is-type? operation? program? resource? rezzed? runner? upgrade?]]
   [game.core.charge :refer [can-charge charge-ability charge-card]]
   [game.core.checkpoint :refer [fake-checkpoint]]
   [game.core.choose-one :refer [choose-one-helper cost-option]]
   [game.core.cost-fns :refer [install-cost play-cost rez-cost]]
   [game.core.damage :refer [damage]]
   [game.core.def-helpers :refer [all-cards-in-hand* in-hand*?
                                  breach-access-bonus defcard draw-abi drain-credits gain-credits-ability  offer-jack-out
                                  reorder-choice run-any-server-ability run-central-server-ability run-remote-server-ability run-server-ability run-server-from-choices-ability scry tutor-abi with-revealed-hand
                                  make-icon]]
   [game.core.drawing :refer [draw]]
   [game.core.effects :refer [register-lingering-effect]]
   [game.core.eid :refer [complete-with-result effect-completed make-eid
                          make-result]]
   [game.core.engine :refer [checkpoint do-nothing not-used-once? pay register-events register-pending-event
                             resolve-ability trigger-event trigger-event-simult
                             unregister-events unregister-floating-events]]
   [game.core.events :refer [first-event? first-run-event? run-events
                             turn-events run-event-count]]
   [game.core.expose :refer [expose]]
   [game.core.finding :refer [find-cid find-latest]]
   [game.core.flags :refer [any-flag-fn? can-rez? can-trash?
                            clear-all-flags-for-card! clear-run-flag! clear-turn-flag!
                            in-corp-scored? register-run-flag! register-turn-flag! zone-locked?]]
   [game.core.gaining :refer [gain gain-clicks gain-credits lose lose-clicks
                              lose-credits]]
   [game.core.hand-size :refer [corp-hand-size+ hand-size]]
   [game.core.hosting :refer [host]]
   [game.core.ice :refer [all-subs-broken? get-strength pump pump-all-icebreakers
                          update-all-ice update-breaker-strength]]
   [game.core.identities :refer [disable-card disable-identity enable-card
                                 enable-identity]]
   [game.core.initializing :refer [card-init make-card]]
   [game.core.installing :refer [install-as-condition-counter install-locked?
                                 runner-can-install? runner-can-pay-and-install? runner-install]]
   [game.core.link :refer [get-link]]
   [game.core.mark :refer [identify-mark-ability mark-changed-event]]
   [game.core.memory :refer [available-mu]]
   [game.core.moving :refer [as-agenda flip-facedown forfeit mill move
                             swap-ice trash trash-cards]]
   [game.core.payment :refer [can-pay? ->c cost-value x-cost-value]]
   [game.core.play-instants :refer [play-instant]]
   [game.core.prevention :refer [damage-name prevent-damage preventable? prevent-end-run prevent-up-to-n-tags prevent-up-to-n-damage]]
   [game.core.prompts :refer [cancellable clear-wait-prompt]]
   [game.core.props :refer [add-counter add-prop]]
   [game.core.revealing :refer [reveal reveal-loud]]
   [game.core.rezzing :refer [derez get-rez-cost rez]]
   [game.core.runs :refer [bypass-ice can-run-server? gain-next-run-credits get-runnable-zones
                           make-run prevent-access successful-run-replace-breach
                           total-cards-accessed]]
   [game.core.sabotage :refer [sabotage-ability]]
   [game.core.say :refer [system-msg]]
   [game.core.servers :refer [central->name is-central? is-remote? remote->name
                              target-server unknown->kw zone->name
                              zones->sorted-names]]
   [game.core.set-aside :refer [get-set-aside set-aside]]
   [game.core.shuffling :refer [shuffle! shuffle-into-deck shuffle-my-deck! fail-to-find!]]
   [game.core.tags :refer [gain-tags gain-tags-ability lose-tags]]
   [game.core.threat :refer [threat threat-level]]
   [game.core.to-string :refer [card-str]]
   [game.core.toasts :refer [toast]]
   [game.core.update :refer [update!]]
   [game.core.virus :refer [get-virus-counters]]
   [game.macros :refer [continue-ability effect msg req wait-for]]
   [game.utils :refer :all]
   [jinteki.utils :refer :all]
   [jinteki.validator :refer [legal?]]))

(defcard "Indexing"
  {:makes-run true
   :on-play (run-server-ability :rd)
   :events [(successful-run-replace-breach
              {:target-server :rd
               :this-card-run true
               :ability
               {:msg "rearrange the top 5 cards of R&D"
                :waiting-prompt true
                :async true
                :effect (effect
                          (continue-ability
                            state side (let [from (take 5 (:deck corp))]
                              (when (pos? (count from))
                                (reorder-choice :corp :corp from '() (count from) from)))
                            card nil))}})]})
