(ns com.wsscode.pathom.trace
  (:require
    [clojure.spec.alpha :as s]
    [clojure.walk :as walk]
    [com.fulcrologic.guardrails.core :refer [>def >defn >fdef => | <- ?]]
    [#?(:clj  com.wsscode.async.async-clj
        :cljs com.wsscode.async.async-cljs)
     :refer [let-chan]])
  #?(:cljs
     (:require-macros
       [com.wsscode.pathom.trace])))

(defn compute-d3-tree [{::keys                        [relative-timestamp duration children details]
                        :com.wsscode.pathom.core/keys [path]
                        :keys                         [key]}]
  (cond-> {:start    relative-timestamp
           :path     path
           :duration (or duration 0)
           :details  (mapv (fn [{::keys [relative-timestamp duration event]
                                 :as    row}]
                             (let [details (->> (dissoc row ::relative-timestamp ::timestamp ::duration ::event)
                                                (into {} (map (fn [[k v]] [(keyword (name k)) v]))))]
                               (merge {:event    (name event)
                                       :duration (or duration 0)
                                       :start    relative-timestamp}
                                      details)))
                       details)}
    key (assoc :name (str key))
    children (assoc :children
               (into [] (map (comp compute-d3-tree second) children)))))
