(ns malli.instrument
  (:require [clojure.walk :as walk]
            [malli.core :as m]
            [malli.generator :as mg]))

(defn -sequential [x] (cond (set? x) x (sequential? x) x :else [x]))

(defn -schema [v]
  (let [{:keys [malli/schema arglists]} (meta v)]
    (or schema (as-> (seq (keep (comp :malli/schema meta) arglists)) $
                 (when (= (count arglists) (count $)) (cond->> $ (next $) (into [:function])))))))

(defn -collect! [v]
  (let [{:keys [ns name] :as m} (meta v)]
    (when-let [s (-schema v)] (m/-register-function-schema! (-> ns str symbol) name s (m/-unlift-keys m "malli")))))

(defn clj-collect!
  ([] (clj-collect! {:ns *ns*}))
  ([{:keys [ns]}]
   (not-empty (reduce (fn [acc v] (let [v (-collect! v)] (cond-> acc v (conj v)))) #{} (vals (mapcat ns-publics (-sequential ns)))))))
