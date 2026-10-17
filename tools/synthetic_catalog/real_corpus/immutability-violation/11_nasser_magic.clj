(ns ^{:doc "Socket server support"
      :author "Alex Miller"}
  clojure.core.server
  (:require [clojure.string :as str]
            [clojure.edn :as edn]
            [clojure.main :as m])
  (:import [System.Net.Sockets Socket SocketException TcpListener TcpClient] [System.Net Dns IPAddress]))

(def ^:dynamic *session* nil)

(defonce ^:private lock (Object.))

(defonce ^:private servers {})

(defmacro ^:private with-lock
   [lock-expr & body]
  `(let [lockee# ~lock-expr]                   ;;;   ~(with-meta lock-expr {:tag 'java.util.concurrent.locks.ReentrantLock})
      (monitor-enter lockee#)                  ;;;  (.lock lockee#)
	  (try
        ~@body
        (finally
          (monitor-exit lockee#)))))

(defmacro ^:private thread
  [name daemon & body]
  `(doto (System.Threading.Thread.  ^System.Threading.ThreadStart (gen-delegate System.Threading.ThreadStart [] ~@body))          ;;; (doto (Thread. (fn [] ~@body) ~name)
    (.set_Name ~name) (.set_IsBackground ~daemon)                                                   ;;; (.setDaemon ~daemon)
    (.Start)))

(defn- accept-connection
  "Start accept function, to be invoked on a client thread, given:
    conn - client socket
    name - server name
    client-id - client identifier
    in - in stream
    out - out stream
    err - err stream
    accept - accept fn symbol to invoke
    args - to pass to accept-fn"
  [^TcpClient conn name client-id in out err accept args]                                 ;;; ^Socket
  (try  (.set_AutoFlush ^System.IO.StreamWriter out true)                                 ;;; Added .set_Autoflush command
    (binding [*in* in
              *out* out
              *err* err
              *session* {:server name :client client-id}]
      (with-lock lock
        (alter-var-root #'servers assoc-in [name :sessions client-id] {}))
      (require (symbol (namespace accept)))
      (let [accept-fn (resolve accept)]
        (apply accept-fn args)))
    (catch SocketException _disconnect)
    (finally
      (with-lock lock
        (alter-var-root #'servers update-in [name :sessions] dissoc client-id))
      (.Close ^System.IO.TextReader in) (.Close conn))))
