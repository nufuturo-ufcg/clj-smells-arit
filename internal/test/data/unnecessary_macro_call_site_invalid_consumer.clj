(ns unnecessary-macro-call-site.invalid-consumer
  (:require [unnecessary-macro-call-site.provider :as provider]))

(provider/add-one)
(provider/add-one value)
