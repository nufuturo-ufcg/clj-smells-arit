(ns braid.core.client.store
  (:require
   [braid.core.client.invites.schema :as invites]
   [braid.core.common.schema :as app-schema]
   [clojure.spec.alpha :as s]
   [spec-tools.data-spec :as ds]))

(def initial-state
  (merge
    {:login-state :gateway ; :ws-connect :app
     :websocket-state {:connected? false
                       :next-reconnect nil}
     :action nil
     :action-disabled? false
     :csrf-token nil
     :user-auth nil
     :open-group-id nil
     :threads {}
     :group-threads {}
     :tags {}
     :groups {}
     :page {:type nil}
     :session nil
     :preferences {}
     :notifications {:window-visible? true
                     :unread-count 0}
     :user {:open-thread-ids #{}
            :subscribed-tag-ids #{}}
     :focused-thread-id nil}
    invites/init-state))
