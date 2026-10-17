(ns puppetlabs.services.analytics.analytics-service
  (:require [clojure.tools.logging :as log]
            [puppetlabs.trapperkeeper.core :refer [defservice]]
            [puppetlabs.i18n.core :as i18n]
            [puppetlabs.dujour.version-check :as version-check]
            [puppetlabs.services.analytics.dropsonde :refer [run-dropsonde]]))

(defprotocol AnalyticsService
  "Protocol placeholder for the analytics service.")
