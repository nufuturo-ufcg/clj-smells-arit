(ns keycloak.utils
  (:require    [clojure.string :as string]
               [clojure.java.io :as io]
               [clojure.pprint :as ppr]
               [me.raynes.fs :as fs])
  (:import [java.net Socket InetSocketAddress]))

(defn server-listening?
  "Check if a given host is listening on a given port in the limit of timeout-ms (default 500 ms)"
  ([host port]
   (server-listening? host port 500))
  ([host port timeout-ms]
   (try (let [socket (Socket.)]
          (.connect socket (InetSocketAddress. host port) timeout-ms))
        (catch Exception e
          (throw (ex-info (format "Host %s at port %s is not listening! (timeout was %d ms)" host port timeout-ms) {:host host :port port :timeout-ms timeout-ms}))))))
