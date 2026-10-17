(ns clojure.core.matrix.impl.defaults
  "Default implementations for core.matrix protocols

   These should be correct reference implementations for all protocols that work on
   arbitrary objects. They are not necessarily tuned for performance.

   Default implementations are defined for:
    - nil (treated as a scalar nil value)
    - Numbers (treated as scalar numerical values)
    - Arbitrary arrays for which the protocol is not otherwise defined
  "
  (:require [clojure.core.matrix.protocols :as mp]
            [clojure.core.matrix.impl.wrappers :as wrap]
            [clojure.core.matrix.impl.mathsops :as mops :refer [to-degrees* to-radians*]]
            [clojure.core.matrix.implementations :as imp]
            [clojure.core.matrix.impl.double-array :as da]
            [clojure.core.matrix.impl.common :refer [logistic-fn softplus-fn relu-fn construct-matrix
                                                     square? symmetric-matrix-entries? mapmatrix]]
            [clojure.core.matrix.utils :as u])
  #?@(:clj [(:require
              [clojure.core.matrix.macros :refer [TODO error scalar-coerce c-for doseq-indexed array?]]
              [clojure.core.matrix.macros-clj :refer [try-current-implementation eps== native-array?]])
            (:import [clojure.lang ISeq])]
      :cljs [(:require-macros
               [clojure.core.matrix.impl.defaults :refer [def-PMathsFunctions def-PMathsFunctionsMutable]]
               [clojure.core.matrix.macros :refer [TODO error scalar-coerce c-for doseq-indexed array?]]
               [clojure.core.matrix.macros-cljs :refer [try-current-implementation eps== native-array?]])]))

(defn update-qr [^doubles qr-data idx mcols mrows ^doubles vs
                 ^doubles us gamma tau]
  (let [idx (long idx)
        mrows (long mrows)
        mcols (long mcols)
        u (aget us idx)
        idx+1 (inc idx)
        gamma (double gamma)
        tau (double tau)]
    (c-for [i idx+1 (< i mcols) (inc i)]
      (aset vs i (aget qr-data
                       (+ i
                          (* idx mcols)))))
    (c-for [i idx+1 (< i mrows) (inc i)]
      (let [qr-idx (+ idx+1
                      (* i mcols))]
        (c-for [j idx+1 (< j mcols) (inc j)]
          (aset vs j
                (+ (aget vs j)
                   (* (aget us i)
                      (aget qr-data (+ qr-idx
                                       (- j idx+1)))))))))
    (c-for [i idx+1 (< i mcols) (inc i)]
      (aset vs i (* (aget vs i)
                    gamma)))

    (c-for [i idx (< i mrows) (inc i)]
      (let [u (aget us i)]
        (c-for [j idx+1 (< j mcols) (inc j)]
          (let [qr-idx (+ (* i mcols)
                          idx+1
                          (- j idx+1))]
            (aset qr-data qr-idx
                  (- (aget qr-data qr-idx) (* u (aget vs j))))))))

    (when (< idx mcols)
      (aset qr-data (+ idx (* idx mcols)) (double (- tau))))

    (c-for [i idx+1 (< i mrows) (inc i)]
      (aset qr-data
            (+ idx (* i mcols))
            (aget us i)))
    {:qr-data qr-data
     :vs vs}))
