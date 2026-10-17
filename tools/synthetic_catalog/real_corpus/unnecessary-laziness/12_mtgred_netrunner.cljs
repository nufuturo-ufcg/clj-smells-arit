(ns nr.end-of-game-stats
  (:require [nr.translations :refer [tr tr-side tr-element]]
            [nr.utils :refer [map-longest]]))

(defn optional-stat
  [s stat-tr key]
  (let [val (get-in s key)]
    (when (and val (pos? val))
      [stat-tr (get-in s key)])))

(defn corp-stats [s]
  (vec (remove nil?
               [[[:stats_clicks-gained "Clicks Gained"] (get-in s [:gain :click])]
                [[:stats_credits-gained "Credits Gained"] (get-in s [:gain :credit])]
                [[:stats_credits-spent "Credits Spent"] (get-in s [:spent :credit])]
                [[:stats_credits-click "Credits by the Basic Action"] (get-in s [:click :credit])]
                [[:stats_cards-drawn "Cards Drawn"] (get-in s [:gain :card])]
                [[:stats_cards-click "Cards Drawn by the Basic Action"] (get-in s [:click :draw])]
                [[:stats_damage-done "Damage Done"] (get-in s [:damage :all])]
                [[:stats_cards-rezzed "Cards Rezzed"] (get-in s [:cards :rezzed])]
                (optional-stat s [:stats_shuffle-count "Shuffle Count"] [:shuffle-count])
                (optional-stat s [:stats_operations-played "Operations Played"] [:cards-played :play-instant])
                (optional-stat s [:stats_rashida-count "Rashida Count"] [:rashida-count])
                ;; psi games
                (optional-stat s [:stats_psi-game-total "Psi Game: Games Played"] [:psi-game :games-played])
                (optional-stat s [:stats_psi-game-total-wins "Psi Game: Wins"] [:psi-game :wins])
                (optional-stat s [:stats_psi-game-total-bid-0 "Psi Game: Bid 0"] [:psi-game :bet-0])
                (optional-stat s [:stats_psi-game-total-bid-1 "Psi Game: Bid 1"] [:psi-game :bet-1])
                (optional-stat s [:stats_psi-game-total-bid-2 "Psi Game: Bid 2"] [:psi-game :bet-2])
                ])))
