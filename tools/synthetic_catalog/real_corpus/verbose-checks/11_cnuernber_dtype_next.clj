(ns tech.v3.datatype.base
  (:require [tech.v3.datatype.protocols :as dtype-proto]
            [tech.v3.datatype.dispatch :as dispatch]
            [tech.v3.datatype.packing :as packing]
            [tech.v3.datatype.errors :as errors]
            [tech.v3.datatype.casting :as casting]
            [tech.v3.parallel.for :as parallel-for]
            [tech.v3.datatype.argtypes :as argtypes]
            [ham-fisted.defprotocol :refer [extend extend-type extend-protocol]]
            [ham-fisted.language :refer [cond]])
  (:import [tech.v3.datatype Buffer BinaryBuffer
            ObjectBuffer ElemwiseDatatype ObjectReader NDBuffer
            LongReader DoubleReader]
           [clojure.lang IPersistentCollection APersistentMap APersistentVector
            APersistentSet IPersistentMap]
           [java.util RandomAccess List Spliterator$OfDouble Spliterator$OfLong
            Spliterator$OfInt Arrays]
           [java.util.stream Stream DoubleStream LongStream IntStream]
           [ham_fisted Reductions Casts])
  (:refer-clojure :exclude [extend extend-type extend-protocol cond]))

(defn shape-ary->stride-ary
  "Strides assuming everything is increasing and packed."
  ^longs [shape-vec]
  (let [retval (long-array (count shape-vec))
        shape-vec (->buffer shape-vec)
        n-elems (alength retval)
        n-elems-dec (dec n-elems)]
    (loop [idx n-elems-dec
           last-stride 1]
      (if (>= idx 0)
        (let [next-stride (* last-stride
                             (if (== idx n-elems-dec)
                               1
                               (.readLong shape-vec (inc idx))))]
          (aset retval idx next-stride)
          (recur (dec idx) next-stride))
        retval))))
