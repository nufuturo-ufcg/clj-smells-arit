(ns codax.store
  (:require
   [clojure.core.cache :as cache]
   [clojure.java.io :as io]
   [taoensso.nippy :as nippy])
  (:import
   [java.io RandomAccessFile]
   [java.nio ByteBuffer]
   [java.nio.channels FileChannel]
   [java.nio.file Files Paths StandardCopyOption StandardOpenOption]
   [java.time Instant ZoneId]
   [java.time.format DateTimeFormatter]
   [java.util.concurrent.locks ReentrantReadWriteLock ReentrantLock]))

(def order 256)

(def cache-threshold 64)

(def nippy-options {:compressor nippy/lz4-compressor})

(def ^:dynamic *monitor-metrics* nil)

(def file-type-tag (long 14404350))

(def file-version-tag (int 1))

(defn- to-path [str]
  (Paths/get str (make-array String 0)))

(defn- open-file-handles [data path]
  (assoc data
         :manifest-channel (FileChannel/open (to-path (str path "/manifest")) (into-array [StandardOpenOption/APPEND
                                                                                           StandardOpenOption/SYNC]))
         :nodes-channel (FileChannel/open (to-path (str path "/nodes")) (into-array [StandardOpenOption/APPEND
                                                                                     StandardOpenOption/SYNC]))
         :file-reader (RandomAccessFile. (str path "/nodes") "r")))

(defn- close-file-handles [db]
  (let [data @(:data db)]
    (.close ^FileChannel (:manifest-channel data))
    (.close ^FileChannel (:nodes-channel data))
    (.close ^RandomAccessFile (:file-reader data))))

(defn- initialize-database-data! [{:keys [path data] :as database} manifest nodes-offset]
  (swap! data
         #(-> %
              (open-file-handles path)
              (assoc :cache (cache/lru-cache-factory {} :threshold cache-threshold)
                     :manifest manifest
                     :writes-since-compaction 0
                     :nodes-offset nodes-offset)))
  database)

(defmacro with-compaction-lock [[database] & body]
  `(let [^ReentrantReadWriteLock lock# (:compaction-lock ~database)]
     (.lock (.writeLock lock#))
     (try
       (do ~@body)
       (finally (.unlock (.writeLock lock#))))))

(defmacro with-write-lock [[database] & body]
  `(let [[^ReentrantLock lock#] (:write-lock ~database)]
     (.lock lock#)
     (try
       (do ~@body)
       (finally (.unlock lock#)))))

(defn- compact-manifest [path manifest root-id]
  (let [compact-manifest-buffer (ByteBuffer/allocate (* 16 (+ 2 (count manifest))))
        compact-manifest-channel (FileChannel/open (to-path (str path "/manifest_COMPACT")) (into-array [StandardOpenOption/CREATE_NEW
                                                                                                         StandardOpenOption/WRITE
                                                                                                         StandardOpenOption/SYNC]))]
    (doto compact-manifest-buffer
      (.putLong file-type-tag)
      (.putInt file-version-tag)
      (.putInt order))
    (loop [remaining-manifest manifest]
      (when-let [[id address] (first remaining-manifest)]
        (doto compact-manifest-buffer
          (.putLong id)
          (.putLong address))
        (recur (rest remaining-manifest))))
    (doto compact-manifest-buffer
      (.putLong 0)
      (.putLong root-id)
      (.flip))
    (doto compact-manifest-channel
      (.write compact-manifest-buffer)
      (.close))))

(defn- compact-nodes [path manifest]
  (let [nodes-channel (FileChannel/open (to-path (str path "/nodes")) (into-array [StandardOpenOption/READ]))
        compact-nodes-channel (FileChannel/open (to-path (str path "/nodes_COMPACT")) (into-array [StandardOpenOption/CREATE_NEW
                                                                                                   StandardOpenOption/APPEND]))]
    (loop [address 0
           new-manifest (transient {})
           remaining-manifest manifest]
      (if-let [[id original-address] (first remaining-manifest)]
        (do
          (.position nodes-channel ^long original-address)
          (let [size-buf (ByteBuffer/allocate 8)
                _ (.read nodes-channel size-buf)
                size (.getLong ^ByteBuffer (.flip size-buf))]
            (.write compact-nodes-channel ^ByteBuffer (.flip size-buf))
            (.transferFrom compact-nodes-channel nodes-channel (+ 8 address) size)
            (recur (+ 8 size address)
                   (assoc! new-manifest id address)
                   (rest remaining-manifest))))
        (do
          (.write compact-nodes-channel (ByteBuffer/allocate 8)) ; 8 empty bytes indicate the end of a commit
          (.force compact-nodes-channel true)
          (.close compact-nodes-channel)
          (.close nodes-channel)
          {:new-manifest (persistent! new-manifest)
           :new-nodes-offset (+ 8 address)})))))

(defn- move-file-atomically [from to]
    (Files/move (to-path from) (to-path to) (into-array [StandardCopyOption/ATOMIC_MOVE])))

(def ts-formatter (.withZone (DateTimeFormatter/ofPattern "YYYYMMdd'T'HHmmss'Z'") (ZoneId/of "UTC")))

(defn- relocate-compact-files [path backup-fn]
  (if backup-fn
    (let [ts-suffix (str "_" (.format ts-formatter (Instant/now)) "_" (System/nanoTime))]
      (move-file-atomically (str path "/nodes") (str path "/nodes" ts-suffix))
      (move-file-atomically (str path "/manifest") (str path "/manifest" ts-suffix))
      (future (backup-fn {:dir path
                          :suffix ts-suffix
                          :file-names [(str "nodes" ts-suffix) (str "manifest" ts-suffix)]})))

    (do
      (move-file-atomically (str path "/nodes") (str path "/nodes_ARCHIVE"))
      (move-file-atomically (str path "/manifest") (str path "/manifest_ARCHIVE"))))
  (move-file-atomically (str path "/nodes_COMPACT") (str path "/nodes"))
  (move-file-atomically (str path "/manifest_COMPACT") (str path "/manifest")))

(defn update-metrics! [db start-compaction-time end-compaction-time writes-since-compaction old-manifest-size old-nodes-size]
  (when *monitor-metrics*
    (let [metrics @(:metrics db)
          new-nodes-size (.size (-> db :data deref :nodes-channel))
          last-compacted-at (or (:compacted-at metrics) (:opened-at metrics))
          compaction-number (inc (:compactions metrics))]
      (swap! (:metrics db) assoc :compacted-at end-compaction-time :compactions compaction-number)
      (when *monitor-metrics* (*monitor-metrics* {:new-manifest-size (* 16 (+ 2 (-> db :data deref :manifest count)))
                                                  :old-manifest-size old-manifest-size
                                                  :new-nodes-size new-nodes-size
                                                  :old-nodes-size old-nodes-size
                                                  :compaction-time (- end-compaction-time start-compaction-time)
                                                  :operation-time (- start-compaction-time last-compacted-at)
                                                  :total-open-time (- end-compaction-time (:opened-at metrics))
                                                  :compaction-number compaction-number
                                                  :writes-since-compaction writes-since-compaction})))))

(defn compact-database [db]
  (with-write-lock [db]
    (let [start-compaction-time (System/nanoTime)
          path (:path db)
          {:keys [manifest root-id is-closed writes-since-compaction ^FileChannel manifest-channel ^FileChannel nodes-channel]} @(:data db)
          _ (when is-closed (throw (ex-info "Database Closed" {:cause :attempted-compaction
                                                               :message "The database object has been invalidated."
                                                               :path path})))
          old-manifest-size (.size manifest-channel)
          old-nodes-size (.size nodes-channel)
          {:keys [new-manifest new-nodes-offset]} (compact-nodes path manifest)]
      (compact-manifest path new-manifest root-id)
      (with-compaction-lock [db]
        (close-file-handles db)
        (relocate-compact-files path (:backup-fn db))
        (initialize-database-data! db new-manifest new-nodes-offset)
        (update-metrics! db start-compaction-time (System/nanoTime) writes-since-compaction old-manifest-size old-nodes-size))
      true)))

(defn- update-database! [{:keys [db root-id id-counter manifest]} nodes-offset manifest-delta dirty-ids nodes-by-address]
  (swap! (:data db)
         (fn [data]
           (let [updated-cache (reduce-kv (fn [c address node]
                                            (let [old-address (manifest (:id node))]
                                              (-> c
                                                  (cache/evict old-address)
                                                  (cache/miss address node))))
                                          (:cache data)
                                          nodes-by-address)]
             (-> data
                 (assoc :root-id root-id
                        :cache updated-cache
                        :id-counter id-counter
                        :writes-since-compaction (inc (:writes-since-compaction data))
                        :nodes-offset nodes-offset)
                 (update :manifest #(apply dissoc % dirty-ids))
                 (update :manifest merge manifest-delta)))))
  (let [^FileChannel manifest-channel (-> db :data deref :manifest-channel)
        size-of-file-manifest (/ (.size manifest-channel) 16)
        auto-compaction-factor (/ (Math/pow size-of-file-manifest 1.2) (+ 2 (count manifest)))]
    (when (> auto-compaction-factor 25)
      (compact-database db))))

(defn- save-buffers! [db ^ByteBuffer manifest-buffer ^ByteBuffer nodes-buffer]
  (let [data @(:data db)
        ^FileChannel manifest-channel (:manifest-channel data)
        ^FileChannel nodes-channel (:nodes-channel data)]
    (.write manifest-channel ^ByteBuffer (.flip manifest-buffer))
    (.write nodes-channel ^ByteBuffer (.flip nodes-buffer))))

(defn commit! [txn]
  (let [dirty-ids (keys (:dirty-nodes txn))
        dirty-nodes (remove (comp nil? second) (:dirty-nodes txn))
        manifest-buffer (ByteBuffer/allocate (* 16 (inc (count dirty-nodes))))]
    (loop [remaining-nodes dirty-nodes
           address (:nodes-offset txn)
           manifest-delta {}
           node-buffers []
           nodes-by-address {}
           total-length 0]
      (if (empty? remaining-nodes)
        (let [all-nodes-buffer (ByteBuffer/allocate (+ 8 total-length))]
          (doseq [^ByteBuffer b node-buffers]
            (.put all-nodes-buffer (.array b)))
          (.putLong all-nodes-buffer (long 0)) ;; 8 empty bytes indicate the end of a commit
          (.putLong manifest-buffer (long 0)) ;; 8 empty bytes indicate the end of a commit
          (.putLong manifest-buffer (long (:root-id txn)))
          (save-buffers! (:db txn) manifest-buffer all-nodes-buffer)
          (update-database! txn (+ 8 address) manifest-delta dirty-ids nodes-by-address))
        (let [[id node] (first remaining-nodes)
              ^bytes encoded-value (nippy/freeze node nippy-options)
              size (count encoded-value)
              buf (ByteBuffer/allocate (+ 8 size))]
          (.putLong buf (long size))
          (.put buf encoded-value)
          (.putLong manifest-buffer (long id))
          (.putLong manifest-buffer (long address))
          (recur (rest remaining-nodes)
                 (+ 8 address size)
                 (assoc manifest-delta id address)
                 (conj node-buffers buf)
                 (assoc nodes-by-address address node)
                 (+ 8 size total-length))))))
  nil)

(defn make-transaction [database touched-paths]
  (let [{:keys [manifest root-id id-counter nodes-offset is-closed]} @(:data database)]
    (when is-closed (throw (ex-info "Database Closed" {:cause :attempted-transaction
                                                       :message "The database object has been invalidated."
                                                       :path (:path database)})))
    {:codax.store/is-transaction true
     :db database
     :root-id root-id
     :id-counter id-counter
     :nodes-offset nodes-offset
     :manifest manifest
     :touched-paths touched-paths
     :dirty-nodes {}}))

(defn assert-db [db & [msg]]
  (when (not (and (map? db) (:codax.store/is-database db)))
    (throw (ex-info "Invalid Database" {:cause :invalid-database
                                        :message (or msg "expected a database")}))))

(defn assert-txn [txn & [msg]]
  (when (not (and (map? txn) (:codax.store/is-transaction txn)))
    (throw (ex-info "Invalid Transaction" {:cause :invalid-transaction
                                           :message (or msg "expected a transaction")
                                           :got txn}))))

(defn update-touched-paths! [tx all-watched-paths]
  (let [touched-paths (:touched-paths tx)]
    (doseq [watched-path-set @all-watched-paths]
      (swap! watched-path-set into touched-paths))))

(defn throw-upgrade-invalidated-transaction []
  (throw (ex-info "Transaction Invalidated By Upgrade" {:cause :transaction-invalidated-by-upgrade
                                                        :message "Once a transaction is upgraded the previous tx object is invalid. If an exception was caught because the `:throw-on-error` option the upgraded tx can be found in the ex-data of ExceptionInfo at the `:codax/upgraded-transaction` key. For example: `(let [tx (:codax/upgraded-transaction (ex-data e)] ...)`"})))

(defmacro with-upgradable-transaction [[database tx-symbol throw-on-upgrade] & body]
  `(let [db# ~database]
     (assert-db db#)
     (let [[^ReentrantLock write-lock# all-watched-paths#] (:write-lock db#)
           ^ReentrantReadWriteLock compaction-lock# (:compaction-lock ~database)
           read-lock# (.readLock compaction-lock#)
           watched-paths# (atom [])
           active-lock# (atom :read)]
       (try
         (do
           (.lock read-lock#)
           (swap! all-watched-paths# conj watched-paths#)
           (let [~tx-symbol (assoc (make-transaction db# [])
                                   :upgrade-watched-paths watched-paths#
                                   :upgrade-viewed-paths (atom [])
                                   :upgrade-lock active-lock#)
                 result-tx# (do ~@body)]
             (assert-txn result-tx#  "the body of `with-upgradable-transaction` must evaluate to a transaction")
             (when (identical? @active-lock# :write)
               (when (:upgrade-viewed-paths result-tx#)
                 (throw-upgrade-invalidated-transaction))
               (commit! result-tx#)
               (update-touched-paths! result-tx# all-watched-paths#))))
         (catch clojure.lang.ExceptionInfo e#
           (if (and (:codax/upgraded-transaction (ex-data e#)) (not ~throw-on-upgrade))
             (with-write-lock [db#]
               (let [~tx-symbol (:codax/upgraded-transaction (ex-data e#))
                     result-tx# (do ~@body)]
                 (assert-txn result-tx#  "the body of `with-upgradable-transaction` must evaluate to a transaction")
                 (commit! result-tx#)
                 (update-touched-paths! result-tx# all-watched-paths#)))
             (throw e#)))
         (finally
           (swap! all-watched-paths# disj watched-paths#)
           (case @active-lock#
             :read (.unlock read-lock#)
             :write (.unlock write-lock#)
             nil))))
     nil))
