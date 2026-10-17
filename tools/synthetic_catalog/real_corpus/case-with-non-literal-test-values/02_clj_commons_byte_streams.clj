(ns
 ^{:deprecated true
   :doc "DEPRECATED: moved to clj-commons.byte-streams.char-sequence"
   :no-doc true
   :superseded-by "clj-commons.byte-streams.char-sequence"}
 byte-streams.char-sequence
  (:refer-clojure :exclude [flush])
  (:import
   [java.util.concurrent.locks
    ReentrantLock]
   [java.io
    ByteArrayOutputStream]
   [java.nio
    ByteBuffer
    CharBuffer]
   [java.nio.charset
    Charset
    CharsetDecoder
    CoderResult
    CodingErrorAction]))

(defn coding-error-action [action]
  (case
   :report  CodingErrorAction/REPORT
   :ignore  CodingErrorAction/IGNORE
   :replace CodingErrorAction/REPLACE))
