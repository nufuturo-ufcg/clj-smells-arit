(ns precept.core
    (:require [precept.util :as util]
              [precept.listeners :as l]
              [precept.state :refer [fact-id rules store state] :as s]
              [clara.rules :refer [fire-rules]]
              [precept.serialize.facts :as serialize]
              [precept.orm :as orm]
              [precept.spec.core :refer [validate]]
              [precept.spec.sub :as sub]
              [precept.spec.lang :as lang]
              [taoensso.sente :as sente]
      #?(:clj
              [clojure.core.async :refer [<! >! put! take! chan go go-loop] :as async])
      #?(:cljs [cljs.core.async :refer [put! take! chan <! >!] :as async])
      #?(:cljs [reagent.core :as r])
        [precept.state :as state])
  #?(:cljs (:require-macros [cljs.core.async.macros :refer [go go-loop]])))

(defn batch-complete?
  "Returns true if the event is fire rules complete or it exists in the batch and
  the maximum event number received is equal to `:total-events` from fire rules complete"
  [event batch]
  (let [max-event-number-recd (max (:event-number event)
                                   (apply max (map :event-number batch)))
        fire-rules-complete-event (or (first (filter :fire-rules-complete batch))
                                      (when (some-> :fire-rules-complete event)
                                        event))
        total-events (:total-events fire-rules-complete-event)]
    (and (integer? total-events)
         (integer? max-event-number-recd)
         (= total-events max-event-number-recd))))

(defn empty-batch?
  [event batch]
  (and (= event {:fire-rules-complete true :total-events -1})
       (empty? batch)))

(defn create-dto>socket-router
  "Returns a go-loop that takes from a channel with events emitted by
  `precept.listeners/PersistentSessionEventMessager and calls the provided send function,
  intended for use with a precept-devtools socket. Batches events per call to `fire-rules`."
  [in send! encoding]
  (go-loop [batch []]
    (let [event (<! in)]
      (cond
        ;; A state might only be comprised of events that were ignored.
        ;; However, we still want to send it to ensure orm-state numbers line up.
        (empty-batch? event batch)
        (do (send! [:devtools/update
                    {:encoding encoding
                     :payload (serialize/serialize encoding batch)}])
            (recur []))

        (batch-complete? event batch)
        (do (send! [:devtools/update
                    {:encoding encoding
                     :payload (serialize/serialize encoding batch)}])
            (recur []))

        :default
        (recur (conj batch event))))))

(defn connect-devtools-socket!
  [options init-cb]
  (let [{:keys [chsk ch-recv send-fn state]} (sente/make-channel-socket!
                                               (:path options)
                                               {:host (:host options)
                                                :type :auto})]
    (def chsk       chsk)
    (def ch-chsk    ch-recv)
    (def chsk-send! send-fn)
    (def chsk-state state)
    (def devtools-socket-ch (chan 5000))
    (def devtools-socket-router (create-dto>socket-router
                                  devtools-socket-ch send-fn (:encoding options)))

    (swap! s/*devtools assoc
      :event-sink devtools-socket-ch
      :host (:host options)
      :path (:path options)
      :encoding (:encoding options))

    (defmulti handle-message first)
    (defmethod handle-message :chsk/ws-ping [_])

    (defmethod handle-message :devtools/init [[msg-name msg]]
      (println "[precept-devtools] Server sent: " msg))

    (defmulti handle-event :id)
    (defmethod handle-event :chsk/handshake [_])

    (defmethod handle-event :chsk/state [{:keys [?data]}]
      (let [[last-state this-state] ?data]
        (when (:first-open? this-state)
          (do (println (str "[precept-devtools] Connected to  "
                            (:host @s/*devtools) (:path @s/*devtools) "."))
              (swap! s/*devtools assoc :connected? true)
              (init-cb (:event-sink @s/*devtools) send-fn)))))

    (defmethod handle-event :chsk/recv [{:keys [?data]}]
      (handle-message ?data))

    (sente/start-chsk-router! ch-recv handle-event)))
