(ns xtdb.pgwire
  (:require [clojure.java.io :as io]
            [clojure.string :as str]
            [clojure.tools.logging :as log]
            [cognitect.anomalies :as-alias anom]
            [cognitect.transit :as transit]
            [integrant.core :as ig]
            [xtdb.authn :as authn]
            [xtdb.db-catalog :as db]
            [xtdb.error :as err]
            [xtdb.expression :as expr]
            [xtdb.metrics :as metrics]
            [xtdb.node :as xtn]
            [xtdb.pgwire.io :as pgio]
            [xtdb.serde :as serde]
            [xtdb.sql :as sql]
            [xtdb.types :as types]
            [xtdb.util :as util]
            [xtdb.vector.writer :as vw]
            [clojure.test :as t]
            [clojure.core :as c])
  (:import io.micrometer.core.instrument.Counter
           [java.io Closeable DataInputStream EOFException IOException PushbackInputStream]
           [java.lang Thread$State]
           [java.net InetAddress ServerSocket Socket SocketException SocketTimeoutException]
           [java.nio ByteBuffer]
           [java.nio.charset StandardCharsets]
           [java.nio.channels FileChannel]
           [java.nio.file Path]
           [java.security KeyStore]
           [java.time Clock Duration ZoneId]
           [java.util.concurrent ExecutorService Future$State FutureTask LinkedBlockingQueue RejectedExecutionException ThreadPoolExecutor TimeUnit]
           [javax.net.ssl KeyManagerFactory SSLContext]
           (org.apache.arrow.memory BufferAllocator)
           org.apache.arrow.vector.types.pojo.Field
           (xtdb.api Authenticator DataSource DataSource$ConnectionBuilder OAuthResult ServerConfig Xtdb Xtdb$Config Xtdb$Connection Xtdb$Statement Xtdb$ExecutedTx Xtdb$SubmittedTx)
           xtdb.api.module.XtdbModule
           (xtdb.arrow Relation Relation$ILoader VectorType)
           xtdb.arrow.RelationReader
           xtdb.database.Database$Config
           (xtdb.api.error Incorrect Interrupted)
           xtdb.api.ResultCursor
           (xtdb.pgwire PgType PgTypes)
           xtdb.JsonSerde
           xtdb.JsonLdSerde
           xtdb.NodeBase
           (xtdb.query SqlParser SqlPlanner
                       ParsedStatement ParsedStatement$Visitor ParsedStatement$Begin
                       ParsedStatement$Commit ParsedStatement$CommitMode ParsedStatement$Query ParsedStatement$Dml
                       ParsedStatement$ShowVariable ParsedStatement$CopyIn ParsedStatement$Execute)
           (xtdb.tx TxOp$PutDocs)))

(defn- xtify-args [_conn args stmt]
  (into [] (map-indexed (->xtify-arg stmt) args)))
