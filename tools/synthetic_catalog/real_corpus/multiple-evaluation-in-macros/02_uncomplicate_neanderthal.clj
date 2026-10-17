(ns ^{:author "Dragan Djuric"}
    uncomplicate.neanderthal.internal.cpp.mkl.factory
  (:refer-clojure :exclude [abs])
  (:require [uncomplicate.commons
             [core :refer [with-release let-release info Releaseable release view]]
             [utils :refer [dragan-says-ex with-check generate-seed]]]
            [uncomplicate.fluokitten.core :refer [fmap! extract]]
            [uncomplicate.clojure-cpp :as cpp
             :refer [pointer long-ptr long-pointer float-pointer double-pointer byte-pointer int-pointer]]
            [uncomplicate.neanderthal
             [core :refer [dim mrows ncols matrix-type entry symmetric?]]
             [real :as real]
             [integer :as integer]
             [math :refer [f=] :as math]
             [block :refer [stride contiguous?]]]
            [uncomplicate.neanderthal.internal
             [constants :refer :all]
             [api :refer :all]
             [navigation :refer [full-storage accu-layout dostripe-layout]]
             [common :refer [check-stride check-eq-navigators flip-uplo real-accessor]]]
            [uncomplicate.neanderthal.internal.cpp
             [common :refer :all]
             [structures :refer :all]
             [blas :refer :all]
             [lapack :refer :all]
             [factory :refer :all :exclude [cast-stream]]]
            [uncomplicate.neanderthal.internal.cpp.mkl
             [constants :refer [mkl-sparse-request]]
             [core :refer [malloc! free! mkl-sparse sparse-matrix mkl-sparse-copy sparse-error
                           dynamic! max-threads max-threads!]]
             [structures :refer [ge-csr-matrix spmat descr sparse-transpose sparse-layout csr-ge-sp2m]]])
  (:import java.nio.ByteBuffer
           [uncomplicate.neanderthal.internal.api DataAccessor Vector LayoutNavigator Region
            GEMatrix UploMatrix DenseStorage]
           [org.bytedeco.javacpp FloatPointer DoublePointer LongPointer IntPointer ShortPointer BytePointer]
           [org.bytedeco.mkl.global mkl_rt mkl_rt$VSLStreamStatePtr]))

(defmacro mkl-real-gb-blas* [name t ptr cpp-ptr cast blas lapack]
  `(extend-type ~name
     Blas
     (swap [_# a# b#]
       (gb-map ~blas ~(cblas t 'swap) ~ptr a# b#)
       a#)
     (copy [_# a# b#]
       (gb-map ~lapack ~(cblas t 'copy) ~ptr a# b#)
       b#)
     (dot [_# a# b#]
       (gb-dot ~blas ~(cblas t 'dot) ~ptr a# b#))
     (nrm1 [_# a#]
       (gb-lan ~lapack ~(lapacke "" t 'langb_64) ~(cblas 'cblas_i t 'amax) ~ptr ~cpp-ptr \O a#))
     (nrm2 [_# a#]
       (gb-lan ~lapack ~(lapacke "" t 'langb_64) ~(cblas t 'nrm2) ~ptr ~cpp-ptr \F a#))
     (nrmi [_# a#]
       (gb-lan ~lapack ~(lapacke "" t 'langb_64) ~(cblas 'cblas_i t 'amax) ~ptr ~cpp-ptr \I a#))
     (asum [_# a#]
       (gb-sum ~blas ~(cblas t 'asum) ~ptr a#))
     (scal [_# alpha# a#]
       (gb-scal ~blas ~(cblas t 'scal) ~ptr (~cast alpha#) a#))
     (axpy [_# alpha# a# b#]
       (gb-axpy ~blas ~(cblas t 'axpy) ~ptr (~cast alpha#) a# b#))
     (mv
       ([_# alpha# a# x# beta# y#]
        (gb-mv ~blas ~(cblas t 'gbmv) ~ptr (~cast alpha#) a# x# (~cast beta#) y#))
       ([_# a# _#]
        (dragan-says-ex "In-place mv! is not supported for GB matrices." {:a (info a#)})))
     (rk
       ([_# _# _# _# a#]
        (dragan-says-ex "rk! is not supported for GB matrices." {:a (info a#)}))
       ([_# _# _# a#]
        (dragan-says-ex "rk! is not supported for GB matrices." {:a (info a#)})))
     (mm
       ([_# alpha# a# b# _#]
        (gb-mm a#))
       ([_# alpha# a# b# beta# c# left#]
        (gb-mm ~blas ~(cblas t 'gbmv) ~ptr (~cast alpha#) a# b# (~cast beta#) c# left#)))))
