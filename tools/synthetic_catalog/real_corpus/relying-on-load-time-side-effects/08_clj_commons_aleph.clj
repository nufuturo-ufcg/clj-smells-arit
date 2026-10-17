(ns aleph.http.client-middleware
  "This middleware is adapted from clj-http, whose license is amenable to this sort of
   copy/pastery"
  (:require
    [aleph.http.common :as common]
    ;; leave this dependency to make sure that HeaderMap is already compiled
    [aleph.http.core :as http]
    [aleph.http.schema :as schema]
    [clj-commons.byte-streams :as bs]
    [clojure.edn :as edn]
    [clojure.string :as str]
    [clojure.walk :refer [prewalk]]
    [manifold.deferred :as d]
    [manifold.executor :as ex]
    [manifold.stream :as s]
    [potemkin :as p])
  (:import
    (aleph.http.core
      HeaderMap)
    (io.netty.buffer
      ByteBuf
      Unpooled)
    (io.netty.handler.codec Headers)
    (io.netty.handler.codec.base64 Base64)
    (io.netty.handler.codec.http
      HttpHeaderNames
      HttpHeaders)
    (io.netty.handler.codec.http.cookie
      ClientCookieDecoder
      ClientCookieEncoder
      DefaultCookie)
    (java.io
      ByteArrayInputStream
      ByteArrayOutputStream
      InputStream)
    (java.net
      IDN
      URL
      URLDecoder
      URLEncoder)
    (java.nio.charset StandardCharsets)))

(def ^:no-doc transit-enabled?
  (try
    (require 'cognitect.transit)
    true
    (catch Throwable _ false)))
