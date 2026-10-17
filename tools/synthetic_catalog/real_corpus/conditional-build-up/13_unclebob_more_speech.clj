(ns more-speech.nostr.event-composers
  (:require [clojure.core.async :as async]
            [clojure.string :as string]
            [more-speech.bech32 :as bech32]
            [more-speech.config :as config :refer [get-db]]
            [more-speech.db.gateway :as gateway]
            [more-speech.logger.default :refer [log-pr]]
            [more-speech.mem :refer :all]
            [more-speech.nostr.contact-list :as contact-list]
            [more-speech.nostr.elliptic-signature :as ecc]
            [more-speech.nostr.events :as events]
            [more-speech.nostr.util :refer :all]
            [more-speech.nostr.util :as util])
  (:import (ecdhJava SECP256K1)))

(defn body->event
  "Adds pubkey, created-at, id, and sig to the partially composed body,
  which must include kind, tags, and content.  The body is put into an
  EVENT wrapper that is ready to send."
  ([body]
   (body->event body
                (get-mem [:keys :private-key])
                (get-mem [:keys :public-key])))

  ([body private-key public-key]
   (let [private-key (util/hex-string->bytes private-key)
         now (util/get-now)
         body (assoc body :pubkey public-key
                          :created_at now)
         [id body] (events/make-id-with-pow config/proof-of-work-default body)
         aux-rand (util/num->bytes 32 (biginteger (System/currentTimeMillis)))
         signature (ecc/do-sign id private-key aux-rand)
         event (assoc body :id (util/bytes->hex-string id)
                           :sig (util/bytes->hex-string signature))]
     ["EVENT" event])))

(defn compose-metadata-event []
  (let [key-map (get-mem :keys)
        profile {:name (:name key-map)
                 :about (:about key-map)
                 :picture (:picture key-map)}
        profile (if (some? (:nip05 key-map))
                  (assoc profile :nip05 (:nip05 key-map))
                  profile)
        profile (if (some? (:lud16 key-map))
                  (assoc profile :lud16 (:lud16 key-map))
                  profile)
        content (events/to-json profile)
        body {:kind 0
              :tags []
              :content content}]
    (body->event body)))
