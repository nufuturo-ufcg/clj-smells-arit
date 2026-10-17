(ns nr.gameboard.board
  (:require-macros [cljs.core.async.macros :refer [go]])
  (:require
   [cljc.java-time.duration :as duration]
   [cljc.java-time.instant :as inst]
   [cljc.java-time.temporal.chrono-unit :as chrono]
   [cljs.core.async :refer [<! chan put!] :as async]
   [clojure.string :as s :refer [capitalize ends-with? join lower-case split
                           starts-with?]]
   [game.core.card :refer [active? asset? corp? facedown? faceup?
                           get-counters get-title has-subtype? ice? program? rezzed?
                           same-card? operation? condition-counter?]]
   [jinteki.cards :refer [all-cards]]
   [jinteki.utils :refer [add-cost-to-label is-tagged? select-non-nil-keys
                          str->int] :as utils]
   [nr.appstate :refer [app-state current-gameid]]
   [nr.cardbrowser :refer [card-as-text]]
   [nr.end-of-game-stats :refer [build-game-stats]]
   [nr.gameboard.actions :refer [send-command]]
   [nr.gameboard.card-preview :refer [card-highlight-mouse-out
                                      card-highlight-mouse-over card-preview-mouse-out
                                      card-preview-mouse-over put-game-card-in-channel zoom-channel]]
   [nr.gameboard.player-stats :refer [stat-controls stats-view]]
   [nr.gameboard.replay :refer [replay-panel]]
   [nr.gameboard.right-pane :refer [content-pane]]
   [nr.gameboard.state :refer [game-state not-spectator? replay-side]]
   [nr.sounds :refer [update-audio]]
   [nr.translations :refer [tr tr-data tr-game-prompt tr-side tr-element tr-span]]
   [nr.utils :refer [banned-span card-colors-class card-colors-custom-style
                     checkbox-button cond-button get-image-path
                     image-or-face map-longest render-icons render-message]]
   [nr.ws :as ws]
   [jinteki.card-backs :as card-backs]
   [reagent.core :as r]))

(defonce card-menu (r/atom {}))

(defn close-card-menu []
  (swap! card-menu dissoc :source :keep-menu-open))

(defn encounter-info-div
  "Displays encounter information including current ice strength and subroutines"
  [ice]
  (let [subtypes (sort-by #(case %
                             "Mythic" 1
                             ("Barrier" "Code Gate" "Sentry") 2
                             ("Bioroid" "Trap") 3
                             4) (:subtypes ice))
        current-strength (or (:current-strength ice)
                             (:strength ice)
                             0)
        subroutines (:subroutines ice)]
    [:div.panel.blue-shade.encounter-info {:style {:display "inline"}}
     [:span.active.float-center (get-title ice)]
     [:span.info {:style {:display "block"}} (join " - " subtypes)]
     [tr-element :span.float-center [:card-browser_strength "Strength"] {:strength current-strength}]
     [:hr]
     (when (seq subroutines)
       [:span.float-center [tr-span [:game_subs "Subroutines"]] ":"])
     (doall
       (map-indexed
         (fn [i sub]
           (let [fire-sub #(when (= :corp (:side @game-state))
                             (send-command "subroutine" {:card ice
                                                         :subroutine i})
                             (close-card-menu))]
             [:div {:key i
                    :tab-index 0
                    :on-click fire-sub
                    :on-key-down #(when (= "Enter" (.-key %))
                                    (fire-sub))
                    :on-key-up #(when (= " " (.-key %))
                                  (fire-sub))}
              [:span (cond (:broken sub)
                           {:class :disabled
                            :style {:font-style :italic}}
                           (false? (:resolve sub))
                           {:class :dont-resolve
                            :style {:text-decoration :line-through}})
               (render-icons (str " [Subroutine] " (:label sub)))]
              [:span.float-right
               (cond (:broken sub) [banned-span]
                     (:fired sub) "✅")]]))
         subroutines))]))

(defn label [cursor opts]
  (let [fn (or (:fn opts) count)
        classes (str (when (pos? (count cursor)) "darkbg ")
                     (:classes opts))]
    [:div.header {:class classes}
     (if (:tr-vec opts) [tr-span (:tr-vec opts) (:tr-params opts)] (:name opts))
     (when-not (:hide-cursor opts) (str " (" (fn cursor) ")"))]))
