(ns io.pithos.stream
  "Read and write to cassandra from OutputStream and InputStream"
  (:import java.io.OutputStream
           java.io.InputStream
           java.nio.ByteBuffer
           org.eclipse.jetty.server.HttpInputOverHTTP
           javax.servlet.ReadListener)
  (:require [io.pithos.blob :as b]
            [io.pithos.desc :as d]
            [io.pithos.util :as u]
            [clojure.tools.logging :refer [debug error]]))

(defn chunk->ba
  "Chunks in pithos come back as bytebuffers and we
   need byte-arrays for outputstreams, this converts
   from the former to the latter.

   The underlying bytebuffers can be reused, which is why
   we need to watch respect the position and limit parameters
   given."
  [{:keys [payload]}]
  (let [array (.array payload)
        off   (.position payload)
        len   (- (.limit payload) off)]
    [array off len]))

(defn stream-file
  "Stream a whole file. Do not handle supplied ranges and
   just write out all chunks."
  [od ^OutputStream stream blob blocks]
  (doseq [{:keys [block]} blocks]
    (doseq [chunk (b/chunks blob od block block)
            :let [[array off len] (chunk->ba chunk)]]
      (.write stream array off len))))
