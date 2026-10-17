(ns leiningen.core.ssl
  (:require [cemerick.pomegranate.aether :as aether]
            [clojure.java.io :as io]
            [leiningen.core.user :as user])
  (:import java.security.KeyStore
           java.security.KeyStore$TrustedCertificateEntry
           java.security.Security
           java.security.cert.CertificateFactory
           javax.net.ssl.KeyManagerFactory
           javax.net.ssl.SSLContext
           javax.net.ssl.TrustManagerFactory
           javax.net.ssl.X509TrustManager
           java.io.FileInputStream
           org.apache.http.config.RegistryBuilder
           org.apache.http.conn.socket.PlainConnectionSocketFactory
           org.apache.http.conn.ssl.DefaultHostnameVerifier
           org.apache.http.conn.ssl.SSLConnectionSocketFactory
           org.apache.http.impl.conn.PoolingHttpClientConnectionManager
           org.apache.maven.wagon.providers.http.HttpWagon))

(defn read-certs
  "Read one or more X.509 certificates in DER or PEM format."
  [f]
  (let [cf (CertificateFactory/getInstance "X.509")
        in (io/input-stream (or (io/resource f) (io/file f)))]
    (.generateCertificates cf in)))
