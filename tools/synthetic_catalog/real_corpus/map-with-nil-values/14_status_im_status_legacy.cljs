(ns status-im.common.home.actions.view
  (:require
    [clojure.string :as string]
    [quo.core :as quo]
    [status-im.common.confirmation-drawer.view :as confirmation-drawer]
    [status-im.common.mute-drawer.view :as mute-drawer]
    [status-im.common.muting.helpers :refer [format-mute-till]]
    [status-im.config :as config]
    [status-im.constants :as constants]
    [status-im.contexts.chat.actions.view :as chat-actions]
    [status-im.contexts.chat.contacts.drawers.nickname-drawer.view :as nickname-drawer]
    [status-im.contexts.communities.actions.chat.view :as communities-chat-actions]
    [utils.i18n :as i18n]
    [utils.re-frame :as rf]))

(defn- entry
  [{:keys [icon label on-press danger? sub-label chevron? add-divider? accessibility-label]}]
  {:pre [(keyword? icon)
         (string? label)
         (fn? on-press)
         (boolean? danger?)
         (boolean? chevron?)]}
  {:icon                icon
   :label               label
   :on-press            on-press
   :danger?             danger?
   :sub-label           sub-label
   :right-icon          (when chevron? :i/chevron-right)
   :add-divider?        add-divider?
   :accessibility-label accessibility-label})

(defn group-details-entry
  [chat-id]
  (entry {:icon                :i/members
          :label               (i18n/label :t/group-details)
          :on-press            (fn []
                                 (rf/dispatch [:chats-list/load-chat chat-id])
                                 (rf/dispatch [:pin-message/load-pin-messages chat-id])
                                 (rf/dispatch [:hide-bottom-sheet])
                                 (rf/dispatch [:navigate-to :screen/group-details chat-id]))
          :danger?             false
          :accessibility-label :group-details
          :sub-label           nil
          :chevron?            false}))
