(ns skuld.flake
  "ID generation. Flake IDs are 160 bits, and comprise:
  
  [64 bits | Timestamp, in milliseconds since the epoch]
  [32 bits | a per-process counter, reset each millisecond]
  [48 bits | a host identifier]
  [16 bits | the process ID]
  
  Note that the timestamp is not Posix time or UTC time. Instead we use the
  JVM's nanoTime, which is a linear time source over intervals smaller than
  ~292 years, and use it to compute an offset from the POSIX time as measured
  *once* by System/currentTimeMillis.

  Regressions in time are not allowed; Flake will periodically serialize the
  current time to disk to prevent regressions."
  (:require [primitive-math :as p])
  (:use     [potemkin :only [deftype+]])
  (:import (java.lang.management ManagementFactory)
           (java.net InetAddress
                     NetworkInterface)
           (java.nio ByteBuffer)
           (java.security MessageDigest)
           (java.util Arrays)
           (java.util.concurrent.atomic AtomicInteger)))

(defn ^"[B" node-id
  "We take all hardware addresses, sort them bytewise, concatenate, hash, and
  truncate to 48 bits."
  []
  (let [addrs (->> (NetworkInterface/getNetworkInterfaces)
                   enumeration-seq
                   (map #(.getHardwareAddress ^NetworkInterface %))
                   (remove nil?))
        md    (MessageDigest/getInstance "SHA-1")]
    (assert (< 0 (count addrs)))
    (doseq [addr addrs]
      (.update md ^bytes addr))
    ; 6 bytes is 48 bits
    (Arrays/copyOf (.digest md) 6)))
