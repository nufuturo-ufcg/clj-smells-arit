(ns macro-call-site.consumer
  (:require [macro-call-site.provider :as provider]))

(provider/duplicated (clojure.core/swap! state inc))
(provider/duplicated 42)
(provider/duplicated external-call)
