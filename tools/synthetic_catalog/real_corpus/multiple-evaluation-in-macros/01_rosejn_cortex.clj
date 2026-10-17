(ns cortex.compute.cpu.tensor-math
  (:require [think.datatype.core :refer [v-aget v-aset] :as dtype]
            [think.datatype.base :as dtype-base]
            [think.datatype.marshal :as marshal]
            [cortex.tensor.math :as tm]
            [clojure.math.combinatorics :as combo]
            [cortex.compute.cpu.driver
             :refer [datatype->view-cast-fn
                     datatype->cast-fn]
             :as cpu-driver]
            [cortex.compute.driver :as compute-drv]
            [think.parallel.core :as parallel]
            [clojure.core.matrix.macros :refer [c-for]]
            [cortex.compute.math-util :as cmu]
            [cortex.compute.driver :as drv]
            [think.resource.core :as resource]
            [cortex.tensor :as ct]
            [cortex.tensor.dimensions :as ct-dims]
            [cortex.nn.impl :as impl]
            [clojure.core.matrix.stats :as stats])
  (:import [cortex.compute.cpu.driver CPUStream]
           [com.github.fommil.netlib BLAS]
           [think.datatype DoubleArrayView FloatArrayView
            LongArrayView IntArrayView ShortArrayView ByteArrayView]
           [java.security SecureRandom]))

(defmacro avg-exc-pad-pooling-forward-impl
  [datatype]
  `(fn [output# input# config#]
     (let [input-ary# (datatype->view-cast-fn ~datatype input#)
           output-ary# (datatype->view-cast-fn ~datatype output#)]
       (impl/convolution-outer-kernel
         config# :pooling
         (impl/convolution-roll-unroll-inner-kernel
           (let [input-val# (datatype->cast-fn ~datatype (if ~'input-valid?
                                        (v-aget input-ary# ~'input-addr)
                                        0.0))
                 output-addr# (+ (* ~'out-y ~'output-width)
                                 ~'out-x
                                 ~'chan-output-offset)
                 output-val# (v-aget output-ary# output-addr#)]
             (v-aset output-ary# output-addr#
                     (+ output-val#
                        (/ input-val#
                           ~'exc-pad-kernel-num-elems)))))))))
