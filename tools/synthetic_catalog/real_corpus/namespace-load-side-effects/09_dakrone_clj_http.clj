(ns clj-http.client
  "Batteries-included HTTP client."
  (:refer-clojure :exclude [get update])
  (:require [clj-commons.slingshot :refer [throw+]]
            [clj-http.conn-mgr :as conn]
            [clj-http.cookies :refer [wrap-cookies]]
            [clj-http.core :as core]
            [clj-http.headers :refer [wrap-header-map]]
            [clj-http.links :refer [wrap-links]]
            [clj-http.util :as util :refer [opt]]
            [clojure.java.io :as io]
            [clojure.stacktrace :refer [root-cause]]
            [clojure.string :as str]
            [clojure.walk :refer [keywordize-keys prewalk]]
            [clojure.xml :as xml])
  (:import [java.io BufferedReader ByteArrayOutputStream EOFException File InputStream]
           [java.net UnknownHostException URL]
           [org.apache.http.entity BufferedHttpEntity ByteArrayEntity FileEntity InputStreamEntity StringEntity]
           [javax.xml.parsers SAXParser SAXParserFactory]
           org.xml.sax.helpers.DefaultHandler
           org.apache.http.impl.conn.PoolingHttpClientConnectionManager
           org.apache.http.impl.nio.conn.PoolingNHttpClientConnectionManager))

(def edn-enabled?
  (try
    (require 'clojure.tools.reader.edn)
    true
    (catch Throwable _ false)))
