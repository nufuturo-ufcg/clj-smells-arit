(ns refs-in-dependency-vector-binding-forms
  (:require [reagent.core :as r]))

(defn let-star-reference []
  (let* [st (r/atom nil)]
    (r/use-effect (fn [] (println @st)) [st])))

(defn if-let-reference []
  (if-let [st (r/atom nil)]
    (r/use-effect (fn [] (println @st)) [st])
    nil))

(defn when-let-reference []
  (when-let [st (r/atom nil)]
    (r/use-effect (fn [] (println @st)) [st])))

(defn shadowed-reference []
  (let [st (r/atom nil)]
    (when-let [st :ordinary]
      (r/use-effect (fn [] (println st)) [st]))))

(defn let-star-shadowed-reference []
  (let* [st (r/atom nil)]
    (let* [st :ordinary]
      (r/use-effect (fn [] (println st)) [st]))))

(defn ordinary-let-star-value []
  (let* [st :ordinary]
    (r/use-effect (fn [] (println st)) [st])))

(defn ordinary-if-let-value []
  (if-let [st :ordinary]
    (r/use-effect (fn [] (println st)) [st])
    nil))

(defn ordinary-when-let-value []
  (when-let [st :ordinary]
    (r/use-effect (fn [] (println st)) [st])))

(defn loop-reference-is-ambiguous []
  (loop [st (r/atom nil)]
    (if true
      (r/use-effect (fn [] (println @st)) [st])
      (recur :ordinary))))
