(ns web.telemetry
  (:require
   [clojure.core.async :refer [<! go timeout]]
   [cljc.java-time.temporal.chrono-unit :as chrono]
   [cljc.java-time.duration :as duration]
   [cljc.java-time.instant :as inst]
   [game.core.board :refer [all-active]]
   [web.app-state :refer [app-state]]
   [web.lobby :refer [lobby-update-uids pool-occupants-info fetch-delay-log!]]
   [web.ws :as ws :refer [connected-sockets connections_]]
   [taoensso.encore :as enc]
   [taoensso.timbre :as timbre]))

(defn- lobby->active-cards
  [lobby]
  (if (:started lobby)
    (let [state (:state lobby)]
      (map :title (concat (all-active state :runner) (all-active state :corp))))
    []))

(defn active-card-frequencies
  [lobbies]
  (frequencies (reduce concat [] (map lobby->active-cards (vals lobbies)))))
