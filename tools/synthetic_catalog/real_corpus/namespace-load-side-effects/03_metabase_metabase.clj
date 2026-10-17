(ns metabase.pulse.send
  "Code related to sending Pulses (Alerts or Dashboard Subscriptions)."
  (:require
   [metabase.models.interface :as mi]
   [metabase.pulse.models.pulse :as models.pulse]
   [metabase.task-history.core :as task-history]
   [metabase.util.cron :as u.cron]
   [metabase.util.log :as log]
   [toucan2.core :as t2]))

(def ^:private send-notification! (requiring-resolve 'metabase.notification.core/send-notification!))
