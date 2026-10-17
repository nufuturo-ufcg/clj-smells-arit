(ns fipp.ednize
  (:require [clojure.string :as s]
            [fipp.util :refer [edn?]]))

(defprotocol IEdn
  "Perform a shallow conversion to an Edn data structure."
  (-edn [x]))

(defprotocol IOverride
  "Mark object as preferring its custom IEdn behavior.")

(extend-protocol IEdn

  UUID
  (-edn [x]
    (tagged-literal 'uuid (str x)))

  ExceptionInfo
  (-edn [x]
    (tagged-literal 'ExceptionInfo
      (merge {:message (ex-message x)
              :data (ex-data x)}
             (when-let [cause (ex-cause x)]
               {:cause cause}))))

  )
