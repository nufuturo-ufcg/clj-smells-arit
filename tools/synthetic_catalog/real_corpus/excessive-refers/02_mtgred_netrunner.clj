(ns game.cards.assets
  (:require
   [clojure.pprint :as pprint]
   [clojure.set :as set]
   [clojure.string :as str]
   [game.core.access :refer [access-bonus access-card get-only-card-to-access installed-access-trigger num-cards-to-access]]
   [game.core.actions :refer [score]]
   [game.core.agendas :refer [update-all-advancement-requirements
                              update-all-agenda-points]]
   [game.core.bad-publicity :refer [gain-bad-publicity lose-bad-publicity]]
   [game.core.board :refer [all-active-installed all-installed all-installed-runner-type get-remotes
                            installable-servers]]
   [game.core.card :refer [agenda? asset? can-be-advanced? corp? event? corp-installable-type?
                           faceup? fake-identity? get-advancement-requirement
                           get-agenda-points get-card get-counters get-title get-zone hardware? has-subtype?
                           has-any-subtype? ice? identity? in-deck? in-discard? in-hand? in-server? installed? is-type?
                           operation? program? resource? rezzed? runner? upgrade?]]
   [game.core.card-defs :refer [card-def]]
   [game.core.checkpoint :refer [fake-checkpoint]]
   [game.core.choose-one :refer [choose-one-helper]]
   [game.core.cost-fns :refer [play-cost]]
   [game.core.damage :refer [damage]]
   [game.core.def-helpers :refer [corp-install-up-to-n-cards corp-recur corp-rez-toast defcard do-meat-damage do-net-damage draw-abi gain-credits-ability give-tags
                                  make-icon place-advancement-counter reorder-choice spend-credits take-credits take-n-credits-ability trash-on-empty get-x-fn with-revealed-hand]]
   [game.core.drawing :refer [draw first-time-draw-bonus max-draw
                              remaining-draws]]
   [game.core.effects :refer [is-disabled-reg? register-lingering-effect update-disabled-cards]]
   [game.core.eid :refer [complete-with-result effect-completed is-basic-advance-action? make-eid get-ability-targets]]
   [game.core.engine :refer [not-used-once? pay register-events resolve-ability trigger-event-sync should-trigger?]]
   [game.core.events :refer [first-event? no-event? turn-events event-count]]
   [game.core.flags :refer [in-corp-scored? in-runner-scored? lock-zone prevent-current
                            prevent-draw
                            register-turn-flag! release-zone when-scored?]]
   [game.core.gaining :refer [gain gain-clicks gain-credits lose lose-clicks
                              lose-credits]]
   [game.core.hand-size :refer [corp-hand-size+ runner-hand-size+]]
   [game.core.hosting :refer [host]]
   [game.core.ice :refer [update-all-ice update-ice-strength]]
   [game.core.initializing :refer [card-init]]
   [game.core.installing :refer [corp-install corp-install-msg]]
   [game.core.moving :refer [as-agenda mill move remove-from-currently-drawing
                             swap-agendas swap-cards swap-installed trash trash-cards]]
   [game.core.optional :refer [get-autoresolve set-autoresolve]]
   [game.core.payment :refer [can-pay? cost-value ->c]]
   [game.core.play-instants :refer [can-play-instant? play-instant]]
   [game.core.prompts :refer [cancellable]]
   [game.core.props :refer [add-counter add-prop set-prop]]
   [game.core.prevention :refer [damage-name preventable? prevent-bad-publicity prevent-damage prevent-expose]]
   [game.core.revealing :refer [reveal]]
   [game.core.rezzing :refer [can-pay-to-rez? derez rez]]
   [game.core.runs :refer [end-run]]
   [game.core.say :refer [play-sfx system-msg]]
   [game.core.servers :refer [is-remote? target-server zone->name]]
   [game.core.set-aside :refer [swap-set-aside-cards]]
   [game.core.shuffling :refer [shuffle! shuffle-into-deck shuffle-my-deck!
                                shuffle-into-rd-effect]]
   [game.core.tags :refer [gain-tags lose-tags]]
   [game.core.threat :refer [threat-level]]
   [game.core.to-string :refer [card-str]]
   [game.core.toasts :refer [toast]]
   [game.core.update :refer [update!]]
   [game.core.winning :refer [check-win-by-agenda win]]
   [game.macros :refer [continue-ability effect msg req wait-for]]
   [game.utils :refer :all]
   [jinteki.utils :refer :all]
   [game.core.link :refer [get-link]]))
