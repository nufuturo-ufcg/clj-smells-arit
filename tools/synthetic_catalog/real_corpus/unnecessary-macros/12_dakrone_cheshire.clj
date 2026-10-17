(ns cheshire.generate
  "Namespace used to generate JSON from Clojure data structures."
  (:import (com.fasterxml.jackson.core JsonGenerator JsonGenerationException)
           (java.util Date Map List Set SimpleTimeZone UUID)
           (java.sql Timestamp)
           (java.text SimpleDateFormat)
           (java.time Instant)
           (java.math BigInteger)
           (java.util.concurrent.atomic AtomicInteger AtomicLong)
           (clojure.lang  Keyword Ratio Symbol)))

(defmacro i?
  "Just to shorten 'instance?' and for debugging."
  [k obj]
  ;;(println :inst? k obj)
  `(instance? ~k ~obj))
