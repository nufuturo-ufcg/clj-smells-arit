(ns more-speech.ui.swing.article-panel
  (:require
    [more-speech.bech32 :as bech32]
    [more-speech.config :refer [get-db]]
    [more-speech.config :as config]
    [more-speech.db.gateway :as gateway]
    [more-speech.mem :refer :all]
    [more-speech.nostr.event-composers :as composers]
    [more-speech.nostr.events :as events]
    [more-speech.nostr.protocol :as protocol]
    [more-speech.nostr.util :as util]
    [more-speech.nostr.zaps :as zaps]
    [more-speech.ui.formatter-util :as formatter-util]
    [more-speech.ui.formatter-util :as f-util]
    [more-speech.ui.formatters :as formatters]
    [more-speech.ui.swing.article-panel-util :as article-panel-util]
    [more-speech.ui.swing.edit-window :as edit-window]
    [more-speech.ui.swing.html-util :as html-util]
    [more-speech.ui.swing.user-info-interface :as html-interface]
    [more-speech.ui.swing.util :as swing-util :refer [copy-to-clipboard]]
    [more-speech.user-configuration :as uconfig]
    )
  (:use (seesaw [border] [core])))

(defn user-name-click [type frame e]
  (let [node (.getComponent e)
        pubkey (config node :user-data)
        hex-id (util/hexify pubkey)
        npub (bech32/encode "npub" pubkey)
        event-id (config (select frame [:#id-label]) :user-data)
        event (gateway/get-event (get-db) event-id)
        profile (gateway/get-profile (get-db) pubkey)
        popup-items [(action :name (str "Copy " (subs hex-id 0 10) "...")
                             :handler (partial copy-to-clipboard hex-id))
                     (action :name (str "Copy " (subs npub 0 10) "...")
                             :handler (partial copy-to-clipboard npub))]
        popup-items (if (some? profile)
                      (conj popup-items
                            (action :name "Get info..."
                                    :handler (fn [_e] (html-interface/show-user-profile pubkey))))
                      popup-items)
        popup-items (if (and (= type :author)
                             (some? profile))
                      (conj popup-items (action :name "Zap author..."
                                                :handler (partial zaps/zap-author event)))
                      popup-items)
        p (popup :items popup-items)]
    (swing-util/show-popup p e)))
