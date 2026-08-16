(ns refs-in-dependency-vector
  (:require [reagent.core :as r]))

(defn named-reference [user-atom]
  (r/use-effect (fn [] (println user-atom)) [user-atom]))

(defn locally-proven-reference []
  (let [st (r/atom nil)]
    (r/use-effect (fn [] (println @st)) [st])))

(defn dereferenced-dependency [user-atom]
  (r/use-effect (fn [] (println @user-atom)) [@user-atom]))

(defn ordinary-value [user-id]
  (r/use-effect (fn [] (println user-id)) [user-id]))
