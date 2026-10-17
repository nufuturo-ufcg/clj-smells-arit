(ns frontend.db.react
  "Transact the tx with some specified relationship so that the components will
  be refreshed when subscribed data changed.
  It'll be great if we can find an automatically resolving and performant
  solution.
  "
  (:require [datascript.core :as d]
            [frontend.date :as date]
            [frontend.db.conn :as conn]
            [frontend.db.utils :as db-utils]
            [frontend.state :as state]
            [frontend.util :as util :refer [react]]
            [cljs.spec.alpha :as s]
            [clojure.core.async :as async]))

(defonce query-state (atom {}))

(def ^:dynamic *query-component* nil)

(def ^:dynamic *reactive-queries* nil)

(defonce query-components (atom {}))

(defn- get-blocks-range
  [result-atom new-result]
  (let [block? (and (coll? new-result)
                    (map? (first new-result))
                    (:block/uuid (first new-result)))]
    (when block?
      {:old [(:db/id (first @result-atom))
             (:db/id (last @result-atom))]
       :new [(:db/id (first new-result))
             (:db/id (last new-result))]})))

(defn set-new-result!
  [k new-result tx-report]
  (when-let [result-atom (get-in @query-state [k :result])]
    (when tx-report
      (when-let [range (get-blocks-range result-atom new-result)]
        (state/set-state! [:ui/pagination-blocks-range (get-in tx-report [:db-after :max-tx])] range)))
    (reset! result-atom new-result)))

(defn kv
  [key value]
  {:db/id -1
   :db/ident key
   key value})

(defn add-q!
  [k query time inputs result-atom transform-fn query-fn inputs-fn]
  (let [time' (int (util/safe-parse-float time))] ;; for robustness. `time` should already be float
    (swap! query-state assoc k {:query query
                                :query-time time'
                                :inputs inputs
                                :result result-atom
                                :transform-fn transform-fn
                                :query-fn query-fn
                                :inputs-fn inputs-fn}))
  result-atom)

(defn add-query-component!
  [key component]
  (when (and key component)
    (swap! query-components update component (fn [col] (set (conj col key))))))

(defn get-query-cached-result
  [k]
  (when-let [result (get @query-state k)]
    (when (satisfies? IWithMeta @(:result result))
      (set! (.-state (:result result))
           (with-meta @(:result result) {:query-time (:query-time result)})))
    (:result result)))

(defn q
  [repo k {:keys [use-cache? transform-fn query-fn inputs-fn disable-reactive?]
           :or {use-cache? true
                transform-fn identity}} query & inputs]
  {:pre [(s/valid? ::react-query-keys k)]}
  (let [kv? (and (vector? k) (= :kv (first k)))
        origin-key k
        k (vec (cons repo k))]
    (when-let [db (conn/get-db repo)]
      (let [result-atom (get-query-cached-result k)]
        (when-let [component *query-component*]
          (add-query-component! k component))
        (when-let [queries *reactive-queries*]
          (swap! queries conj origin-key))
        (if (and use-cache? result-atom)
          result-atom
          (let [{:keys [result time]} (util/with-time
                                        (-> (cond
                                              query-fn
                                              (query-fn db nil nil)

                                              inputs-fn
                                              (let [inputs (inputs-fn)]
                                                (apply d/q query db inputs))

                                              kv?
                                              (db-utils/entity db (last k))

                                              (seq inputs)
                                              (apply d/q query db inputs)

                                              :else
                                              (d/q query db))
                                            transform-fn))
                result-atom (or result-atom (atom nil))]
            ;; Don't notify watches now
            (set! (.-state result-atom) result)
            (if disable-reactive?
              result-atom
              (add-q! k query time inputs result-atom transform-fn query-fn inputs-fn))))))))

(defn- execute-query!
  [graph db k tx {:keys [query query-time inputs transform-fn query-fn inputs-fn result]}
   {:keys [skip-query-time-check?]}]
  (when (or skip-query-time-check?
            (<= (or query-time 0) 80))
    (let [new-result (->
                      (cond
                        query-fn
                        (let [result (query-fn db tx result)]
                          (if (coll? result)
                            (doall result)
                            result))

                        inputs-fn
                        (let [inputs (inputs-fn)]
                          (apply d/q query db inputs))

                        (keyword? query)
                        (db-utils/get-key-value graph query)

                        (seq inputs)
                        (apply d/q query db inputs)

                        :else
                        (d/q query db))
                      transform-fn)]
     (when-not (= new-result result)
       (set-new-result! k new-result tx)))))
