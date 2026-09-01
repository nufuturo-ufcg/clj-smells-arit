(ns macro-call-site.invalid-consumer
  (:require [macro-call-site.provider :as provider]))

(provider/duplicated)
(provider/duplicated (.toString object))
