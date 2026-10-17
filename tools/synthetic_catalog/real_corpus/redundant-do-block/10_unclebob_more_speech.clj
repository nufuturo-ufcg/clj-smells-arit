(ns more-speech.nostr.zaps
  (:require
    [clj-http.client :as client]
    [clojure.core.async :as async]
    [clojure.data.json :as json]
    [clojure.string :as string]
    [more-speech.bech32 :as bech32]
    [more-speech.config :as config]
    [more-speech.config :refer [get-db]]
    [more-speech.db.gateway :as gateway]
    [more-speech.logger.default :refer [log-pr]]
    [more-speech.mem :refer :all]
    [more-speech.nostr.elliptic-signature :as es]
    [more-speech.nostr.event-composers :as composers]
    [more-speech.nostr.events :as events]
    [more-speech.nostr.relays :as relays]
    [more-speech.nostr.util :as util]
    [more-speech.relay :as relay]
    [more-speech.ui.formatter-util :as formatter-util]
    [more-speech.ui.formatters :as formatters]
    [more-speech.util.fortune-messages :as fortune]
    [more-speech.websocket-relay :as ws-relay]
    [org.bovinegenius.exploding-fish :as uri]
    )
  (:use (seesaw [core]))
  (:import (ecdhJava SECP256K1)
           (java.net URLEncoder)))

(defn get-wc-info [event-chan relay msg]
  (condp = (first msg)
    "EOSE" (do
             (relay/send relay ["CLOSE" (second msg)]))
    "EVENT" (do
              (let [event (nth msg 2)]
                (async/>!! event-chan (util/translate-event event))))
    "OK" (do
           (when-not (nth msg 2)
             (log-pr 1 'get-wc-info 'zap-request-failed (::ws-relay/url relay) msg)))

    (do
      (log-pr 1 'get-wc 'unexpected (::ws-relay/url relay) msg)))
  )
