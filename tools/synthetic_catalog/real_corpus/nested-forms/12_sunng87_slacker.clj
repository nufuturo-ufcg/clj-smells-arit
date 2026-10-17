(ns ^:no-doc slacker.client.common
  (:require [clojure.tools.logging :as log]
            [clojure.string :refer [split]]
            [link.core :as link :refer :all]
            [link.tcp :refer :all]
            [slacker.protocol :as protocol]
            [link.codec :refer [netty-encoder netty-decoder]]
            [link.ssl :refer [ssl-handler-from-jdk-ssl-context]]
            [rigui.core :as rigui]
            [slacker.serialization :refer [serialize deserialize]]
            [slacker.common :refer :all]
            [slacker.protocol :refer :all]
            [manifold.deferred :as d])
  (:import [java.net ConnectException InetSocketAddress InetAddress]
           [java.nio.channels ClosedChannelException]
           [java.util.concurrent ExecutorService]
           [clojure.lang IFn IDeref IBlockingDeref IPending]
           [io.netty.buffer ByteBuf]))

(defprotocol SlackerClientFactoryProtocol
  (schedule-task [this task delay] [this task delay interval])
  (shutdown [this])
  (get-purgatory [this addr])
  (get-purgatory-all [this])
  (open-tcp-client [this host port])
  (assoc-client! [this client])
  (dissoc-client! [this client]))

(deftype DefaultSlackerClientFactory [tcp-factory timer purgatory]
  SlackerClientFactoryProtocol
  (schedule-task [this task delay]
    (rigui/later! timer task delay))
  (schedule-task [this task delay interval]
    (rigui/every! timer task delay interval))
  (shutdown [this]
    ;; shutdown associated clients
    (doseq [a (map :refs (vals @purgatory))]
      (doseq [c (flatten a)]
        (close c)))
    (stop-clients tcp-factory)
    (rigui/stop timer))
  (get-purgatory [this addr]
    (@purgatory addr))
  (get-purgatory-all [this]
    @purgatory)
  (open-tcp-client [this host port]
    (tcp-client tcp-factory host port :lazy-connect true))
  (assoc-client! [this client]
    (swap! purgatory
           (fn [snapshot]
             (let [addr (server-addr client)]
               (if (snapshot addr)
                 (update-in snapshot [addr :refs] conj client)
                 (assoc snapshot addr
                        {:pendings (atom {})
                         :idgen (atom 0)
                         :keep-alive (atom {})
                         :refs [client]}))))))
  (dissoc-client! [this client]
    (swap! purgatory
           (fn [snapshot]
             (let [addr (server-addr client)
                   refs (remove #(= client %)
                                (-> snapshot (get addr) :refs))]
               (if (empty? refs)
                 (dissoc snapshot addr)
                 (assoc snapshot :refs refs)))))))
