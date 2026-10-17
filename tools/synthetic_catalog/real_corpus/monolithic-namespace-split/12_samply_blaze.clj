(ns blaze.db.impl.index.tx-success
  "Functions for accessing the TxSuccess index.

  The TxSuccess index contains the real point in time, as java.time.Instant,
  successful transactions happened. In other words, this index maps each t which
  is just a monotonically increasing number to a real point in time."
  (:require
   [blaze.byte-buffer :as bb]
   [blaze.db.impl.index.cbor :as cbor]
   [blaze.db.impl.iterators :as i]
   [blaze.db.kv :as kv])
  (:import
   [com.github.benmanes.caffeine.cache CacheLoader LoadingCache]
   [com.google.common.primitives Longs]
   [java.time Instant]))

(defn- decode-tx [value-bytes t]
  (let [{:keys [inst]} (cbor/read value-bytes)]
    {:blaze.db/t t
     :blaze.db.tx/instant (Instant/ofEpochMilli inst)}))

(defn- encode-key [^long t]
  (Longs/toByteArray t))

(defn cache-loader [kv-store]
  (reify CacheLoader
    (load [_ t]
      (some-> (kv/get kv-store :tx-success-index (encode-key t))
              (decode-tx t)))))
