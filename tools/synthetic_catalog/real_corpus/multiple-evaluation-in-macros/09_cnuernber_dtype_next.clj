(ns tech.v3.datatype.io-concat-buffer
  (:require [tech.v3.datatype.casting :as casting]
            [tech.v3.datatype.protocols :as dtype-proto]
            [tech.v3.datatype.errors :as errors]
            [clj-commons.primitive-math :as pmath])
  (:import [java.util List]
           [tech.v3.datatype Buffer]
           [ham_fisted ChunkedList]))

(defn- as-prim-io ^Buffer [item] item)

(defmacro ^:private gen-read-macro
  [idx n-elems read-method _n-buffers buffers]
  `(do (errors/check-idx ~idx ~n-elems)
       (loop [buf-idx# 0
              idx# ~idx]
         (let [buffer# (as-prim-io (.get ~buffers buf-idx#))
               buf-len# (.lsize buffer#)]
           (if (< idx# buf-len#)
             (~read-method (as-prim-io buffer#) idx#)
             (recur (unchecked-inc buf-idx#) (pmath/- idx# buf-len#)))))))
