(ns clojure.core.matrix.impl.common
  "Namespace containing common functions useful for core.matrix implementatations"
  (:require [clojure.string :as s]
            [clojure.core.matrix.protocols :as mp]
            [clojure.core.matrix.utils :as u]
            [clojure.core.matrix.implementations :as mi]
    #?(:clj [clojure.java.shell :refer [sh]]))
  (#?(:clj :require :cljs :require-macros)
           [clojure.core.matrix.macros :refer [scalar-coerce]]))

(defn construct-matrix
  "Constructs an array from the provided data, attempting to use the given implementation.
   Uses a default implementation if needed"
  ([impl data]
    (or (mp/construct-matrix impl data)
        (try 
          (mp/construct-matrix mi/*matrix-implementation* data)
          (catch #?(:clj ClassCastException
                   :cljs js/object) t nil)) ;; fix for element type not handled
        (mp/construct-matrix [] data))))

(defn mapmatrix
  "Maps a function over all components of a persistent vector matrix. Like mapv but for matrices.
   Assumes correct dimensionality / shape.

   First array argument must be nested persistent vectors. Others may be
   any arrays of the same shape.

   Returns a nested persistent vector matrix or a scalar value."
  ([f m]
   (let [dims (long (mp/dimensionality m))]
     (cond
       (== 0 dims) (f (scalar-coerce m))
       (== 1 dims) (construct-matrix m
                     (map f (mp/element-seq m)))
       :else
       (let [res (map (partial mapmatrix f) (mp/get-major-slice-seq m))]
         (mp/reshape (mp/coerce-param m res)
                     (vec (concat [(first (mp/get-shape m))] (next (mp/get-shape res)))))))))
  ([f m1 m2]
    (let [dims (long (mp/dimensionality m1))]
      (cond
        (== 0 dims) (f (scalar-coerce m1) (scalar-coerce m2))
        (== 1 dims) (construct-matrix m1
                      (map f (mp/element-seq m1) (mp/element-seq m2)))
        :else
        (let [res (map (partial mapmatrix f) (mp/get-major-slice-seq m1) (mp/get-major-slice-seq m2))]
          (mp/reshape (mp/coerce-param m1 res)
            (vec (concat [(first (mp/get-shape m1))] (next (mp/get-shape res)))))))))
  ([f m1 m2 m3]
    (let [dims (long (mp/dimensionality m1))]
      (cond
        (== 0 dims) (f (scalar-coerce m1) (scalar-coerce m2) (scalar-coerce m3))
        (== 1 dims) (construct-matrix m1
                      (map f (mp/element-seq m1) (mp/element-seq m2) (mp/element-seq m3)))
        :else
        (let [res (mapv (partial mapmatrix f)
                        (mp/get-major-slice-seq m1)
                        (mp/get-major-slice-seq m2)
                        (mp/get-major-slice-seq m3))]
          (mp/reshape (mp/coerce-param m1 res)
            (vec (concat [(first (mp/get-shape m1))] (next (mp/get-shape res)))))))))
  ([f m1 m2 m3 & more]
    (let [dims (long (mp/dimensionality m1))]
      (cond
        (== 0 dims) (apply f (scalar-coerce m1) (scalar-coerce m2)
                           (scalar-coerce m3) (map mp/get-0d more))
        (== 1 dims) (construct-matrix m1
                      (apply map f (mp/element-seq m1) (mp/element-seq m2)
                             (mp/element-seq m3) (map mp/element-seq more)))
        :else
        (let [res (apply map (partial mapmatrix f)
                         (mp/get-major-slice-seq m1)
                         (mp/get-major-slice-seq m2)
                         (mp/get-major-slice-seq m3)
                         (map mp/get-major-slice-seq more))]
          (mp/reshape (mp/coerce-param m1 res)
            (vec (concat [(first (mp/get-shape m1))] (next (mp/get-shape res))))))))))
