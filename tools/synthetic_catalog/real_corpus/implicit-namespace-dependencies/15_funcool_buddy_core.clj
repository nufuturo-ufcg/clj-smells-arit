(ns buddy.core.dsa
  "Digital Signature Algorithms."
  (:refer-clojure :exclude [resolve])
  (:require [buddy.core.codecs :refer :all]
            [clojure.java.io :as io])
  (:import java.security.PublicKey
           java.security.PrivateKey
           java.security.Signature
           java.security.Security))

(def ^:no-doc
  +algorithms+
  {:rsassa-pss+sha256    #(Signature/getInstance "SHA256withRSAandMGF1" "BC")
   :rsassa-pss+sha384    #(Signature/getInstance "SHA384withRSAandMGF1" "BC")
   :rsassa-pss+sha512    #(Signature/getInstance "SHA512withRSAandMGF1" "BC")
   :rsassa-pkcs15+sha256 #(Signature/getInstance "SHA256withRSA" "BC")
   :rsassa-pkcs15+sha384 #(Signature/getInstance "SHA384withRSA" "BC")
   :rsassa-pkcs15+sha512 #(Signature/getInstance "SHA512withRSA" "BC")
   :ecdsa+sha256         #(Signature/getInstance "SHA256withECDSA" "BC")
   :ecdsa+sha384         #(Signature/getInstance "SHA384withECDSA" "BC")
   :ecdsa+sha512         #(Signature/getInstance "SHA512withECDSA" "BC")
   :eddsa                #(Signature/getInstance "EDDSA" "BC")})

(defn- resolve
  "Given dynamic type engine, try resolve it to
  valid engine instance. By default accepts keywords
  and functions."
  [signer]
  (cond
    (keyword? signer)
    (when-let [factory (get +algorithms+ signer)]
      (factory))
    (fn? signer)
    (signer)))
