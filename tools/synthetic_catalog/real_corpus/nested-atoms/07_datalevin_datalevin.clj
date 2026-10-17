(ns ^:no-doc pod.huahaiy.datalevin
  "Implement babashka pod"
  (:refer-clojure :exclude [sync read read-string])
  (:require
   [bencode.core :as bencode]
   [sci.core :as sci]
   [datalevin.core :as d]
   [datalevin.util :as u]
   [datalevin.lmdb :as l]
   [datalevin.interpret :as i]
   [datalevin.protocol :as p]
   [datalevin.bits :as b]
   [datalevin.csv :as csv]
   [datalevin.datom :as dd]
   [datalevin.db :as db]
   [datalevin.interface :as if]
   [datalevin.storage :as st]
   [clojure.java.io :as io]
   [clojure.walk :as w])
  (:import
   [java.io PushbackInputStream]
   [java.nio.charset StandardCharsets]
   [datalevin.storage Store]
   [datalevin.entity Entity]
   [datalevin.db DB]
   [datalevin.search SearchEngine]
   [datalevin.vector VectorIndex]
   [java.util UUID])
  (:gen-class))

(defonce ^:private dl-conns (atom {}))

(defonce ^:private wdl-conns (atom {}))

(defonce ^:private dl-dbs (atom {}))

(defonce ^:private wdl-dbs (atom {}))

(defn- get-cn [{:keys [::conn writing?]}]
  (if writing? (get @wdl-conns conn) (get @dl-conns conn)))

(defn- db-ref
  ([db]
   (db-ref db false))
  ([db writing?]
   (let [id (UUID/randomUUID)]
     (swap! (if writing? wdl-dbs dl-dbs) assoc id db)
     (cond-> {::db id}
       writing? (assoc :writing? true)))))

(defn- rp->res
  ([rp]
   (rp->res rp false))
  ([rp include-dbs?]
   (cond-> {:tx-data (:tx-data rp)
            :tempids (:tempids rp)
            :tx-meta (:tx-meta rp)}
     include-dbs?
     (assoc :db-before (db-ref (:db-before rp))
            :db-after (db-ref (:db-after rp))))))

(defn transact!
  ([cn tx-data]
   (transact! cn tx-data nil))
  ([{:keys [::conn writing?] :as cn} tx-data tx-meta]
   (when-let [c (get-cn cn)]
     (let [rp (try
                (d/transact! c tx-data tx-meta)
                (catch Exception e
                  (when (:resized (ex-data e))
                    (if writing?
                      (let [outer-c  (get @dl-conns conn)
                            outer-db ^DB @outer-c
                            outer-s  (.-store outer-db)
                            write-s  (.-store ^DB @c)
                            write-l  (.-lmdb ^Store write-s)
                            d        (db/transfer outer-db
                                                  (st/transfer outer-s write-l))]
                        (swap! wdl-dbs assoc conn d)
                        (swap! wdl-conns assoc conn
                               (atom d :meta (meta outer-c))))
                      (let [s (.-store ^DB @c)
                            d (db/new-db s)]
                        (swap! dl-dbs assoc conn d)
                        (swap! dl-conns assoc conn
                               (atom d :meta (meta c))))))
                  (throw e)))]
       (rp->res rp)))))

(defn db [{:keys [::conn] :as cn}]
  (when-let [c (get-cn cn)]
    (let [db       (d/db c)
          writing? (:writing? cn)
          dbs      (if writing? wdl-dbs dl-dbs)
          ref      (cond-> {::db conn}
                     writing? (assoc :writing? true))]
      (swap! dbs assoc conn db)
      ref)))
