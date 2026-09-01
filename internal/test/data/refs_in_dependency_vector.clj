(ns refs-in-dependency-vector
  (:require [reagent.core :as r]))

(defn named-reference [user-atom]
  (r/use-effect (fn [] (println user-atom)) [user-atom]))

(defn locally-proven-reference []
  (let [st (r/atom nil)]
    (r/use-effect (fn [] (println @st)) [st])))

(defn shadowed-reference []
  (let [st (r/atom nil)]
    (let [st :ordinary]
      (r/use-effect (fn [] (println st)) [st]))))

(defn dereferenced-dependency [user-atom]
  (r/use-effect (fn [] (println @user-atom)) [@user-atom]))

(defn ordinary-value [user-id]
  (r/use-effect (fn [] (println user-id)) [user-id]))

(defn state-from-use-state [prev-state]
  (r/use-effect (fn [] (println prev-state)) [prev-state]))

;; Negative: a qualified name alone does not prove a mutable reference constructor.
(defn external-atom-name []
  (let [st (example/atom nil)]
    (r/use-effect (fn [] (println st)) [st])))

;; Negative: a qualified name alone does not prove the effect hook contract.
(defn external-effect-hook []
  (let [st (r/atom nil)]
    (example/use-effect (fn [] (println st)) [st])))

(defn locally-proven-ref []
  (let [st (ref nil)]
    (r/use-effect (fn [] (println @st)) [st])))
