(ns ^:no-doc wkok.openai-clojure.sse
  (:require
   [martian.hato :as martian.hato]
   [hato.client :as http]
   [hato.middleware :as hm]
   [clojure.core.async :as a]
   [clojure.string :as string]
   [cheshire.core :as json])
  (:import (java.io InputStream)))

(defn deliver-events
  [events {:keys [on-next]}]
  (when on-next
    (a/go
      (loop []
        (when-let [event (a/<! events)]
          (when (not= :done event)
            (on-next event)
            (recur)))))))
