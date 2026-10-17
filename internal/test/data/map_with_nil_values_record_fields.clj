(ns map-with-nil-values-record-fields)

(defprotocol Lifecycle (start [c]) (stop [c]))

;; A record's base field cannot be omitted: dissoc would return a plain map.
(defrecord Db [conn spec]
  Lifecycle
  (start [this] (assoc this :conn (connect spec)))
  (stop [this] (assoc this :conn nil)))

(defrecord Cache [store])
(extend-type Cache
  Lifecycle
  (start [c] c)
  (stop [c] (assoc c :store nil)))

;; Other maps, and keys that are not base fields, are still reported.
(defrecord Web [server]
  Lifecycle
  (start [this] this)
  (stop [this] (assoc this :extra nil)))

;; m may be a Db: its type is unknown, so a base field is not reported.
(defn stop-map [m] (assoc m :conn nil))

(defn clear-token [m] (assoc m :token nil))
