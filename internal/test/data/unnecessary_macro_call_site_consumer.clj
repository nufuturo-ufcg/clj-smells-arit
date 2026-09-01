(ns unnecessary-macro-call-site.consumer
  (:require [unnecessary-macro-call-site.provider :as provider]))

(provider/add-one 41)
