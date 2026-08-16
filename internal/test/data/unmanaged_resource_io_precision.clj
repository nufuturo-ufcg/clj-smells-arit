(ns unmanaged-resource-io-precision
  (:require
   [clojure.java.io :as io]
   [example.io :as fake-io])
  (:import
   (java.io ByteArrayOutputStream FileInputStream)))

;; Exact namespace alias: locally owned and not closed.
(defn alias-reader-leak [path]
  (let [r (io/reader path)]
    (line-seq r)))

;; Imported Closeable constructor: locally owned and not closed.
(defn imported-stream-leak [path]
  (let [in (FileInputStream. path)]
    (.read in)))

;; A resource wrapper bound locally still owns the wrapped resource.
(defn wrapped-stream-leak [path]
  (let [gzip (java.util.zip.GZIPInputStream. (io/input-stream path))]
    (.read gzip)))

;; Names merely containing resource-related words are not creators.
(defn connection-accessor [context]
  (let [connection (context->app-connection context :search)]
    connection))

(defn system-connections [system]
  (let [configured (system-with-connections system [:search])]
    configured))

(defn byte-stream [buffers]
  (let [body (byte-stream-from-buffers buffers 10)]
    body))

(defn socket-setting [config]
  (let [timeout (socket-timeout config)]
    timeout))

(defn java-setter [pool]
  (let [configured (.setConnectionPoolName pool "main")]
    configured))

;; Unknown consumers may take ownership. High-precision mode suppresses it.
(defn transferred-after-binding [path]
  (let [r (io/reader path)]
    (parser/parse-stream r true)))

;; Direct arguments are also treated as an ownership transfer.
(defn transferred-directly [path]
  (parser/parse-stream (io/reader path) true))

;; Factory functions transfer lifecycle responsibility to the caller.
(defn make-reader [path]
  (io/reader path))

;; A bound resource returned as the value of the let also escapes safely.
(defn make-bound-reader [path]
  (let [r (io/reader path)]
    r))

;; Alternate Java interop close syntax is recognized.
(defn manually-closed [path]
  (let [r (io/reader path)]
    (try
      (line-seq r)
      (finally
        (. r close)))))

;; The alias resolves to another namespace and must not be guessed as java.io.
(defn unrelated-reader [path]
  (let [r (fake-io/reader path)]
    (fake-io/use r)))

;; In-memory resources and wrappers around them own no external descriptor.
(defn in-memory-writer []
  (let [w (io/writer (ByteArrayOutputStream.))]
    (.write w "data")))

;; The in-memory resource may be introduced by an earlier binding.
(defn in-memory-wrapper []
  (let [out (ByteArrayOutputStream.)
        gzip (java.util.zip.GZIPOutputStream. out)]
    (.finish gzip)))

;; The `(new Class ...)` Java syntax is recognized as an exact creator.
(defn new-server-socket-leak [port]
  (let [server (new java.net.ServerSocket port)]
    (.accept server)))

;; Closing the lifecycle root suppresses an adapter over that resource.
(defn socket-adapter-closed [host port]
  (let [socket (new java.net.Socket host port)
        writer (io/writer socket)]
    (.write writer "ping")
    (.close socket)))

;; Non-executed examples must never produce findings.
(comment
  (let [r (io/reader "example.txt")]
    (line-seq r)))
