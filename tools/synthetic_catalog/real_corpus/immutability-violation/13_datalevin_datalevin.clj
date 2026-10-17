(ns datalevin.jepsen.local
  (:require
   [datalevin.constants :as c]
   [datalevin.ha :as dha]
   [datalevin.jepsen.init-cache :as init-cache]
   [datalevin.jepsen.local.cluster :as lcluster]
   [datalevin.jepsen.local.faults :as lfaults]
   [datalevin.jepsen.local.ops :as lops]
   [datalevin.jepsen.local.remote :as lremote]
   [datalevin.jepsen.remote :as remote]
   [datalevin.kv :as kv]
   [datalevin.server :as srv]
   [jepsen.db :as db]))

(def ^:private cluster-timeout-ms 10000)

(def ^:private conn-client-opts {:pool-size 1 :time-out cluster-timeout-ms})

(def default-nodes ["n1" "n2" "n3"])

(defonce ^:private clusters (atom {}))

(defonce ^:private next-port-block (atom -1))

(defonce ^:private remote-runtime-nodes (atom {}))

(def ^:private base-fetch-ha-leader-txlog-batch dha/fetch-ha-leader-txlog-batch)

(def ^:private base-report-ha-replica-floor! dha/report-ha-replica-floor!)

(def ^:private base-fetch-ha-endpoint-snapshot-copy!
  dha/fetch-ha-endpoint-snapshot-copy!)

(def ^:dynamic *remote-launcher-ops* nil)

(defn- remote-cluster?
  [cluster-id]
  (true? (get-in @clusters [cluster-id :remote?])))

(defn- cluster-entry-for-db-identity
  [db-identity]
  (some (fn [[cluster-id {:keys [db-identity] :as cluster}]]
          (when (= db-identity (:db-identity cluster))
            [cluster-id cluster]))
        @clusters))

(defn remote-runtime-node
  [db-identity node-id]
  (get @remote-runtime-nodes [db-identity node-id]))

(defn admin-uri
  [endpoint]
  (lcluster/admin-uri endpoint))

(defn endpoint-for-node
  [cluster-id logical-node]
  (get-in @clusters [cluster-id :node-by-name logical-node :endpoint]))

(defn ^:redef cluster-state
  [cluster-id]
  (get @clusters cluster-id))

(defn- remote-deps
  []
  {:clusters clusters
   :remote-launcher-ops *remote-launcher-ops*
   :endpoint-for-node endpoint-for-node
   :cluster-setup-timeout-ms lcluster/cluster-setup-timeout-ms
   :wait-for-single-leader! (fn [cluster-id timeout-ms]
                              (wait-for-single-leader! cluster-id timeout-ms))
   :transport-failure? transport-failure?
   :admin-uri admin-uri
   :conn-client-opts conn-client-opts})

(defn- faults-deps
  []
  {:clusters clusters
   :remote-runtime-node remote-runtime-node
   :cluster-entry-for-db-identity cluster-entry-for-db-identity
   :endpoint-for-node endpoint-for-node
   :write-remote-content! (fn [ssh node remote-path content]
                            (lremote/write-remote-content! (remote-deps)
                                                           ssh
                                                           node
                                                           remote-path
                                                           content))
   :delete-remote-path! (fn [ssh node remote-path]
                          (lremote/delete-remote-path! (remote-deps)
                                                       ssh
                                                       node
                                                       remote-path))
   :sync-remote-network-state! (fn [cluster-id]
                                 (lremote/sync-remote-network-state!
                                  (assoc (remote-deps)
                                         :endpoint-for-node endpoint-for-node)
                                  cluster-id))
   :wait-for-authority-leader! (fn [cluster-id timeout-ms]
                                 (wait-for-authority-leader! cluster-id
                                                             timeout-ms))
   :cluster-state cluster-state
   :base-fetch-ha-leader-txlog-batch base-fetch-ha-leader-txlog-batch
   :base-report-ha-replica-floor! base-report-ha-replica-floor!
   :base-fetch-ha-endpoint-snapshot-copy!
   base-fetch-ha-endpoint-snapshot-copy!})

(defn storage-fault
  [cluster-id logical-node]
  (lfaults/storage-fault (faults-deps) cluster-id logical-node))

(def transport-failure? lfaults/transport-failure?)

(defn- cluster-deps
  []
  {:clusters clusters
   :next-port-block next-port-block
   :default-nodes default-nodes
   :remote-deps remote-deps
   :fault-deps (fn []
                 {:heal-network! (fn [cluster-id]
                                   (lfaults/heal-network! (faults-deps)
                                                          cluster-id))})
   :wait-for-single-leader! (fn [cluster-id timeout-ms]
                              (wait-for-single-leader! cluster-id timeout-ms))
   :node-diagnostics (fn [cluster-id logical-node]
                       (node-diagnostics cluster-id logical-node))
   :effective-local-lsn (fn [cluster-id logical-node]
                          (effective-local-lsn cluster-id logical-node))
   :transport-failure? transport-failure?})

(defn- ops-deps
  []
  {:clusters clusters
   :remote-cluster? remote-cluster?
   :remote-deps remote-deps
   :cluster-deps cluster-deps
   :transport-failure? transport-failure?
   :storage-fault storage-fault})

(defonce ^:private partition-aware-ha-transports-installed?
  (do
    (alter-var-root
     #'dha/fetch-ha-leader-txlog-batch
     (constantly
      (fn [db-name m leader-endpoint from-lsn upto-lsn]
        (lfaults/partition-aware-fetch-ha-leader-txlog-batch
         (faults-deps) db-name m leader-endpoint from-lsn upto-lsn))))
    (alter-var-root
     #'dha/report-ha-replica-floor!
     (constantly
      (fn [db-name m leader-endpoint applied-lsn]
        (lfaults/partition-aware-report-ha-replica-floor!
         (faults-deps) db-name m leader-endpoint applied-lsn))))
    (alter-var-root
     #'dha/fetch-ha-endpoint-snapshot-copy!
     (constantly
      (fn [db-name m endpoint dest-dir]
        (lfaults/partition-aware-fetch-ha-endpoint-snapshot-copy!
         (faults-deps) db-name m endpoint dest-dir))))
    true))

(defn effective-local-lsn
  [cluster-id logical-node]
  (lops/effective-local-lsn (ops-deps) cluster-id logical-node))

(defn ^:redef wait-for-single-leader!
  ([cluster-id]
   (wait-for-single-leader! cluster-id cluster-timeout-ms))
  ([cluster-id timeout-ms]
   (lops/wait-for-single-leader! (ops-deps) cluster-id timeout-ms)))

(defn ^:redef wait-for-authority-leader!
  ([cluster-id]
   (wait-for-authority-leader! cluster-id cluster-timeout-ms))
  ([cluster-id timeout-ms]
   (lops/wait-for-authority-leader! (ops-deps) cluster-id timeout-ms)))

(defn node-diagnostics
  [cluster-id logical-node]
  (lops/node-diagnostics (ops-deps) cluster-id logical-node))

(defn heal-network!
  [cluster-id]
  (lfaults/heal-network! (faults-deps) cluster-id))
