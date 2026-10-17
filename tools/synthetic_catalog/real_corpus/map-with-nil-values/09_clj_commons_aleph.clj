(ns aleph.http.multipart
  (:require
   [aleph.http.core :as http1]
   [aleph.http.encoding :refer [encode]]
   [aleph.netty :as netty]
   [clj-commons.byte-streams :as bs]
   [clojure.core :as cc]
   [clojure.tools.logging :as log]
   [manifold.deferred :as d]
   [manifold.stream :as s])
  (:import
   [java.util
    Locale
    UUID]
   [java.io
    File]
   [java.nio
    ByteBuffer]
   [java.nio.charset
    Charset]
   [java.net
    URLConnection]
   [io.netty.util.internal
    ThreadLocalRandom]
   [io.netty.handler.codec.http
    DefaultHttpContent
    DefaultHttpRequest
    HttpConstants]
   [io.netty.handler.codec.http.multipart
    Attribute
    MemoryAttribute
    MemoryFileUpload
    FileUpload
    DefaultHttpDataFactory
    HttpPostRequestDecoder
    HttpPostRequestEncoder
    InterfaceHttpData
    InterfaceHttpData$HttpDataType]))

(defmulti ^:no-doc http-data->map
  (fn [^InterfaceHttpData data]
    (.getHttpDataType data)))

(defmethod http-data->map InterfaceHttpData$HttpDataType/Attribute
  [^Attribute attr]
  (let [content (.getValue attr)]
    {:part-name (.getName attr)
     :content content
     :name nil
     :charset (-> attr .getCharset .toString)
     :mime-type nil
     :transfer-encoding nil
     :memory? (.isInMemory attr)
     :file? false
     :file nil
     :size (count content)}))

(defmethod http-data->map InterfaceHttpData$HttpDataType/FileUpload
  [^FileUpload data]
  (let [memory? (.isInMemory data)]
    {:part-name (.getName data)
     :content (when memory?
                (bs/to-input-stream (netty/acquire (.content data))))
     :name (.getFilename data)
     :charset (-> data .getCharset .toString)
     :mime-type (.getContentType data)
     :transfer-encoding (.getContentTransferEncoding data)
     :memory? memory?
     :file? true
     :file (when-not memory? (.getFile data))
     :size (.length data)}))
