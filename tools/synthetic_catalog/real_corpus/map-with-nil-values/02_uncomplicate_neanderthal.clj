(ns ^{:author "Dragan Djuric"}
    uncomplicate.neanderthal.internal.cpp.cuda.factory
  (:refer-clojure :exclude [abs])
  (:require [clojure.java.io :as io]
            [uncomplicate.commons
             [core :refer [Releaseable release let-release with-release info]]
             [utils :refer [with-check dragan-says-ex count-groups generate-seed
                            count-groups scan-resources]]]
            [uncomplicate.fluokitten.protocols :refer [extract]]
            [uncomplicate.clojure-cpp :as cpp :refer [get-entry]]
            [uncomplicate.clojurecuda
             [core :refer :all :exclude [device *headers*]]
             [info :refer [driver-version]]
             [toolbox :refer [launch-reduce! read-int]]]
            [uncomplicate.neanderthal
             [core :refer [dim ncols mrows entry matrix-type]]
             [block :refer [stride offset contiguous? column?]]]
            [uncomplicate.neanderthal.internal
             [api :refer :all]
             [navigation :refer [full-storage diag-unit?]]
             [common :refer [check-eq-navigators]]]
            [uncomplicate.neanderthal.internal.cpp.common :refer :all]
            [uncomplicate.neanderthal.internal.cpp.cuda
             [constants :refer :all]
             [structures :refer :all]]
            [uncomplicate.neanderthal.internal.device.common
             :refer [name-transp uplo-bottom? layout-match? symmetric-match?]])
  (:import [io.github.classgraph ClassGraph ScanResult ResourceList]
           org.bytedeco.cuda.global.cublas
           org.bytedeco.cuda.cublas.cublasContext
           org.bytedeco.cuda.cudart.CUstream_st
           [uncomplicate.neanderthal.internal.api Vector Matrix GEMatrix DataAccessor Region
            DenseStorage FullStorage LayoutNavigator]
           [uncomplicate.neanderthal.internal.cpp.cuda.structures
            CUBlockVector CUGEMatrix CUUploMatrix]))

(defprotocol HandleProvider
  (handle ^cublasContext [this]))

(defn ^:private not-available []
  (throw (UnsupportedOperationException. "Not available in CUDA. Please use a host instance.")))

(defn ^:private vector-equals [modl hstream x y]
  (let [cnt (dim x)]
    (if (< 0 cnt)
      (with-release [equals-kernel (function modl "vector_equals")
                     eq-flag-buf (mem-alloc-driver Integer/BYTES)]
        (memset! eq-flag-buf 0)
        (launch! equals-kernel (grid-1d cnt) hstream
                 (parameters cnt (extract x) (offset x) (stride x) (extract y) (offset y) (stride y)
                             eq-flag-buf))
        (= 0 (read-int hstream eq-flag-buf)))
      (= 0 (dim y)))))

(defn ^:private vector-copy
  ([modl hstream x y kx lx ky]
   (when (< 0 (int lx))
     (with-release [copy-kernel (function modl "vector_copy")]
       (launch! copy-kernel (grid-1d lx) hstream
                (parameters lx (extract x) (* (offset x) (long kx) (stride x)) (stride x)
                            (extract y) (+ (offset y) (* (long ky) (stride y))) (stride y)))))
   y)
  ([modl hstream x y]
   (with-release [copy-kernel (function modl "vector_copy")]
     (launch! copy-kernel (grid-1d (dim x)) hstream
              (parameters (dim x) (extract x) (offset x) (stride x) (extract y) (offset y) (stride y))))
   y))

(defn ^:private vector-swap [modl hstream x y]
  (with-release [copy-kernel (function modl "vector_swap")]
    (launch! copy-kernel (grid-1d (dim x)) hstream
             (parameters (dim x) (extract x) (offset x) (stride x) (extract y) (offset y) (stride y))))
  x)

(defn ^:private vector-sum [modl hstream kernel-name x]
  (let [da (data-accessor x)
        cnt (dim x)
        block-dim 1024]
    (if (< 0 cnt)
      (with-release [sum-kernel (function modl kernel-name)
                     sum-reduction-kernel (function modl "sum_reduction")
                     cu-acc (mem-alloc-driver (* (.entryWidth da) (count-groups block-dim cnt)))
                     res (.wrapPrim da 0.0)]
        (launch-reduce! hstream sum-kernel sum-reduction-kernel
                        [(extract x) (offset x) (stride x) cu-acc] [cu-acc] cnt block-dim)
        (get-entry (memcpy-host! cu-acc res hstream) 0))
      0.0)))

(defn ^:private vector-set [modl hstream alpha x]
  (when (< 0 (dim x))
    (with-release [set-kernel (function modl "vector_set")]
      (launch! set-kernel (grid-1d (dim x)) hstream
               (parameters (dim x) (.castPrim (data-accessor x) alpha) (extract x) (offset x) (stride x)))))
  x)

(defn ^:private vector-axpby [modl hstream alpha x beta y]
  (when (< 0 (dim x))
    (let [da (data-accessor x)]
      (with-release [axpby-kernel (function modl "vector_axpby")]
        (launch! axpby-kernel (grid-1d (dim x)) hstream
                 (parameters (dim x)
                             (.castPrim da alpha) (extract x) (offset x) (stride x)
                             (.castPrim da beta) (extract y) (offset y) (stride y))))))
  y)

(defmacro ^:private vector-scal
  ([cublas-handle cublas method ptr cpp-ptr alpha x]
   `(if (< 0 (dim ~x))
      (with-release [alpha# (~cpp-ptr (.wrapPrim (data-accessor ~x) ~alpha))]
        (with-check cublas-error
          (. ~cublas ~method ~cublas-handle (dim ~x) alpha# (~ptr ~x) (stride ~x))
          ~x))
      ~x))
  ([modl hstream alpha x]
   `(when (< 0 (dim ~x))
      (with-release [set-kernel# (function ~modl "vector_scal")]
        (launch! set-kernel# (grid-1d (dim ~x)) ~hstream
                 (parameters (dim ~x) (.castPrim (data-accessor ~x) ~alpha) (extract ~x) (offset ~x) (stride ~x))))
      ~x)))

(defn ^:private ge-equals [modl hstream a b]
  (if (< 0 (dim a))
    (let [stor (full-storage a)]
      (with-release [equals-kernel (function modl (name-transp "ge_equals" a b))
                     eq-flag-buf (mem-alloc-driver Integer/BYTES)]
        (memset! eq-flag-buf 0)
        (launch! equals-kernel (grid-2d (.sd stor) (.fd stor)) hstream
                 (parameters (.sd stor) (.fd stor) (extract a) (offset a) (.ld stor)
                             (extract b) (offset b) (stride b) eq-flag-buf))
        (= 0 (read-int hstream eq-flag-buf))))
    (= 0 (dim b))))

(defn ^:private ge-set-scal [modl hstream kernel-name alpha a]
  (if (< 0 (dim a))
    (let [da (data-accessor a)
          stor (full-storage a)]
      (with-release [ge-set-scal-kernel (function modl kernel-name)]
        (launch! ge-set-scal-kernel (grid-2d (.sd stor) (.fd stor)) hstream
                 (parameters (.sd stor) (.fd stor) (.castPrim da alpha) (extract a) (offset a) (.ld stor))))))
  a)

(defn ge-copy-swap [modl hstream kernel-name a b]
  (if (< 0 (dim a))
    (let [stor (full-storage a)]
      (with-release [ge-copy-swap-kernel (function modl (name-transp kernel-name a b))]
        (launch! ge-copy-swap-kernel (grid-2d (.sd stor) (.fd stor)) hstream
                 (parameters (.sd stor) (.fd stor) (extract a) (offset a) (.ld stor)
                             (extract b) (offset b) (stride b)))))))

(defn ge-axpby [modl hstream alpha a beta b]
  (if (< 0 (dim a))
    (let [da (data-accessor a)
          stor (full-storage a)]
      (with-release [ge-axpby-kernel (function modl (name-transp "ge_axpby" a b))]
        (launch! ge-axpby-kernel (grid-2d (.sd stor) (.fd stor)) hstream
                 (parameters (.sd stor) (.fd stor) (extract a)
                             (.castPrim da alpha) (offset a) (.ld stor)
                             (.castPrim da beta) (extract b) (offset b) (stride b))))))
  b)

(defn ^:private ge-sum [modl hstream kernel-name a]
  (let [da (data-accessor a)
        stor (full-storage a)
        cnt-sd (.sd stor)
        cnt-fd (.fd stor)
        wgs-sd 32
        wgs-fd 32
        wgs 1024
        grid-dim-x (count-groups wgs-sd cnt-sd)
        grid-dim-y (count-groups wgs-fd cnt-fd)
        acc-count (int (* grid-dim-x grid-dim-y))]
    (if (< 0 (dim a))
      (with-release [sum-kernel (function modl kernel-name)
                     sum-reduction-kernel (function modl "sum_reduction")
                     cu-acc (mem-alloc-driver (* Double/BYTES acc-count))
                     res (.wrapPrim da 0.0)
                     params (parameters acc-count cu-acc)]
        (launch! sum-kernel (grid-2d cnt-sd cnt-fd wgs-sd wgs-fd) hstream
                 (parameters cnt-sd cnt-fd cu-acc (extract a) (offset a) (stride a)))
        (if (< 1 acc-count)
          (launch-reduce! hstream sum-reduction-kernel sum-reduction-kernel
                          params params acc-count wgs))
        (get-entry (memcpy-host! cu-acc res hstream) 0))
      0.0)))

(deftype IntegerVectorEngine [modl hstream]
  BlockEngine
  (equals-block [this x y]
    (vector-equals modl hstream x y))
  Blas
  (swap [_ x y]
    (vector-swap modl hstream x y))
  (copy [_ x y]
    (vector-copy modl hstream x y))
  (dot [_ _  _];TODO
    (not-available))
  (nrm1 [this x]
    (asum this x))
  (nrm2 [_ _]
    (not-available))
  (nrmi [_ _]
    (not-available))
  (asum [_ x]
    (vector-sum modl hstream "vector_asum" x))
  (iamax [_ _];;TODO
    (not-available))
  (iamin [_ _];TODO
    (not-available))
  (rot [_ _ _ _ _]
    (not-available))
  (rotg [_ _]
    (not-available))
  (rotm [_ _ _ _]
    (not-available))
  (rotmg [_ _ _]
    (not-available))
  (scal [_ alpha x]
    (vector-scal modl hstream alpha x)
    x)
  (axpy [_ alpha x y]
    (vector-axpby modl hstream alpha x 1 y))
  BlasPlus
  (amax [_ _];TODO
    (not-available))
  (subcopy [this x y kx lx ky]
    (vector-copy modl hstream x y kx lx ky))
  (sum [_ x]
    (vector-sum modl hstream "vector_sum" x))
  (imax [_ _];TODO
    (not-available))
  (imin [_ _];;TODO
    (not-available))
  (set-all [this alpha x]
    (vector-set modl hstream alpha x))
  (axpby [_ alpha x beta y]
    (vector-axpby modl hstream alpha x beta y)))

(deftype IntegerGEEngine [modl hstream]
  BlockEngine
  (equals-block [_ a b]
    (ge-equals modl hstream a b))
  Blas
  (swap [_ a b]
    (ge-copy-swap modl hstream "ge_copy" a b)
    a)
  (copy [_ a b]
    (ge-copy-swap modl hstream "ge_swap" a b)
    b)
  (dot [_ _  _]
    (not-available))
  (nrm1 [_ _]
    (not-available))
  (nrm2 [_ _]
    (not-available))
  (nrmi [_ _]
    (not-available))
  (asum [_ a]
    (ge-sum modl hstream "ge_asum" a))
  (iamax [_ _]
    (not-available))
  (iamin [_ _]
    (not-available))
  (rot [_ _ _ _ _]
    (not-available))
  (rotg [_ _]
    (not-available))
  (rotm [_ _ _ _]
    (not-available))
  (rotmg [_ _ _]
    (not-available))
  (scal [_ alpha a]
    (ge-set-scal modl hstream "ge_scal" alpha a))
  (axpy [_ alpha a b]
    (ge-axpby modl hstream alpha a 1 b))
  BlasPlus
  (amax [_ _]
    (not-available))
  (subcopy [_ _ _ _ _ _]
    (not-available))
  (sum [_ a]
    (ge-sum modl hstream "ge_sum" a))
  (imax [_ _]
    (not-available))
  (imin [_ _]
    (not-available))
  (set-all [this alpha a]
    (ge-set-scal modl hstream "ge_set" alpha a))
  (axpby [_ alpha a beta b]
    (ge-axpby modl hstream alpha a beta b)))

(deftype FloatVectorEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype DoubleVectorEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype FloatGEEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype DoubleGEEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype FloatTREngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype DoubleTREngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype FloatSYEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype DoubleSYEngine [cublas-handle modl hstream]
  HandleProvider
  (handle [_]
    cublas-handle))

(deftype CUFactory [modl hstream ^DataAccessor da native-fact vector-eng ge-eng tr-eng sy-eng]
  Releaseable
  (release [_]
    (release modl)
    (release da)
    true)
  DataAccessorProvider
  (data-accessor [_]
    da)
  FactoryProvider
  (factory [this]
    this)
  (native-factory [_]
    native-fact)
  FlowProvider
  (flow [_]
    hstream)
  MemoryContext
  (compatible? [_ o]
    (compatible? da o))
  (device [_]
    :cuda)
  RngStreamFactory
  (create-rng-state [_ seed]
    (atom seed))
  Factory
  (create-vector [this n init]
    (let-release [res (cu-block-vector this n)]
      (when init
        (.initialize da (extract res)))
      res))
  (create-ge [this m n column? init]
    (let-release [res (cu-ge-matrix this m n column?)]
      (when init
        (.initialize da (extract res)))
      res))
  (create-uplo [this n mat-type column? lower? diag-unit? init]
    (let-release [res (cu-uplo-matrix this n column? lower? diag-unit? mat-type)]
      (when init
        (.initialize da (extract res)))
      res))
  (create-tr [this n column? lower? diag-unit? init]
    (let-release [res (cu-uplo-matrix this n column? lower? diag-unit?)]
      (when init
        (.initialize da (extract res)))
      res))
  (create-sy [this n column? lower? init]
    (let-release [res (cu-uplo-matrix this n column? lower?)]
      (when init
        (.initialize da (extract res)))
      res))
  (vector-engine [_]
    vector-eng)
  (ge-engine [_]
    ge-eng)
  (tr-engine [_]
    tr-eng)
  (sy-engine [_]
    sy-eng))

(defn ^:private get-cublas-stream [^cublasContext handle]
  (let [res (CUstream_st.)]
    (with-check cublas-error (cublas/cublasGetStream_v2 handle res) res)))

(defn ^:private cublas-handle
  "Creates a cuBLAS context handler on the specific `stream`."
  [^CUstream_st stream]
  (let [handle (cublasContext.)]
    (with-check cublas-error (cublas/cublasCreate_v2 handle)
      (with-check cublas-error (cublas/cublasSetStream_v2 handle stream) handle))))

(let [number-src (apply str
                        (slurp (io/resource "uncomplicate/clojurecuda/kernels/reduction.cu"))
                        (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/number.cu"))
                        (map (comp slurp io/resource) (scan-resources ["uncomplicate/neanderthal/internal/device/cuda/number"])))

      number-standard-src (str number-src
                               (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/vect-math.cu"))
                               (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/number-standard.cu")))

      random-number-src (apply str
                               (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/random-number.cu"))
                               (map (comp slurp io/resource) (scan-resources ["uncomplicate/neanderthal/internal/device/cuda/random"])))

      real-src (apply str number-standard-src random-number-src
                      (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/real.cu"))
                      (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/vect-math-real.cu"))
                      (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/random-real.cu"))
                      (map (comp slurp io/resource) (scan-resources ["uncomplicate/neanderthal/internal/device/cuda/real"])))

      integer-src (apply str number-standard-src
                         (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/integer.cu"))
                         (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/vect-math-integer.cu"))
                         (map (comp slurp io/resource) (scan-resources ["uncomplicate/neanderthal/internal/device/cuda/integer"])))


      half-src (apply str number-src random-number-src
                      (slurp (io/resource "uncomplicate/neanderthal/internal/device/cuda/half.cu"))
                      (map (comp slurp io/resource) (scan-resources ["uncomplicate/neanderthal/internal/device/cuda/half"])))

      standard-headers {"stdint.h" nil
                        "float.h" nil}

      philox-headers (merge standard-headers
                            {"Random123/philox.h"
                             (slurp (io/resource "uncomplicate/neanderthal/internal/device/include/Random123/philox.h"))
                             "features/compilerfeatures.h"
                             (slurp (io/resource "uncomplicate/neanderthal/internal/device/include/Random123/features/compilerfeatures.h"))
                             "nvccfeatures.h"
                             (slurp (io/resource "uncomplicate/neanderthal/internal/device/include/Random123/features/nvccfeatures.h"))
                             "array.h" (slurp (io/resource "uncomplicate/neanderthal/internal/device/include/Random123/array.h"))})

      half-headers (merge philox-headers
                          {"cuda_fp16.h" nil})]

  (defn cublas-double [native-double ctx hstream]
    (in-context
        ctx
      (with-release [prog (compile! (program real-src philox-headers)
                                    ["-DNUMBER=double" "-DREAL=double" "-DACCUMULATOR=double"
                                     "-DCAST(fun)=fun" #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"
                                     (format "-DCUDART_VERSION=%s" (driver-version))])]
        (let-release [modl (module prog)
                      handle (cublas-handle hstream)
                      hstream (get-cublas-stream handle)
                      double-accessor (->DoublePointerAccessor ctx hstream
                                                               (fn [^long size]
                                                                 (cuda-malloc size :double))
                                                               cuda-free!)]
          (->CUFactory modl hstream double-accessor native-double
                       (->DoubleVectorEngine handle modl hstream) (->DoubleGEEngine handle modl hstream)
                       (->DoubleTREngine handle modl hstream) (->DoubleSYEngine handle modl hstream))))))

  (defn cublas-float [native-float ctx hstream]
    (in-context ctx
      (with-release [prog (compile! (program real-src philox-headers)
                                    ["-DNUMBER=float" "-DREAL=float" "-DACCUMULATOR=float"
                                     "-DCAST(fun)=fun##f" #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"
                                     (format "-DCUDART_VERSION=%s" (driver-version))])]
        (let-release [modl (module prog)
                      handle (cublas-handle hstream)
                      hstream (get-cublas-stream handle)
                      float-accessor (->FloatPointerAccessor ctx hstream
                                                             (fn [^long size]
                                                               (cuda-malloc size :float))
                                                             cuda-free!)]
          (->CUFactory modl hstream float-accessor native-float
                       (->FloatVectorEngine handle modl hstream) (->FloatGEEngine handle modl hstream)
                       (->FloatTREngine handle modl hstream) (->FloatSYEngine handle modl hstream))))))

  (defn cublas-half [native-half ctx hstream]
    (in-context ctx

      (with-release [prog (compile! (program half-src half-headers)
                                    ["-DNUMBER=short"
                                     #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"])]
        (let-release [modl (module prog)
                      half-accessor (->HalfPointerAccessor ctx hstream
                                                           (fn [^long size]
                                                             (cuda-malloc size :short))
                                                           cuda-free!)]
          (->CUFactory modl hstream half-accessor native-half
                       (->IntegerVectorEngine modl hstream) (->IntegerGEEngine modl hstream) nil nil)))))

  (defn cublas-long [native-long ctx hstream]
    (in-context ctx
      (with-release [prog (compile! (program integer-src)
                                    ["-DNUMBER=long" "-DINTEGER=long" "-DACCUMULATOR=long"
                                     #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"])]
        (let-release [modl (module prog)
                      handle (cublas-handle hstream)
                      hstream (get-cublas-stream handle)
                      long-accessor (->LongPointerAccessor ctx hstream
                                                           (fn [^long size]
                                                             (cuda-malloc size :long))
                                                           cuda-free!)]
          (->CUFactory modl hstream long-accessor native-long
                       (->IntegerVectorEngine modl hstream) (->IntegerGEEngine modl hstream) nil nil)))))

  (defn cublas-int [native-int ctx hstream]
    (in-context ctx
      (with-release [prog (compile! (program integer-src)
                                    ["-DNUMBER=int" "-DINTEGER=int" "-DACCUMULATOR=int"
                                     #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"])]
        (let-release [modl (module prog)
                      handle (cublas-handle hstream)
                      hstream (get-cublas-stream handle)
                      int-accessor (->IntPointerAccessor ctx hstream
                                                         (fn [^long size]
                                                           (cuda-malloc size :int))
                                                         cuda-free!)]
          (->CUFactory modl hstream int-accessor native-int
                       (->IntegerVectorEngine modl hstream) (->IntegerGEEngine modl hstream) nil nil)))))

  (defn cublas-short [native-short ctx hstream]
    (in-context ctx
      (with-release [prog (compile! (program integer-src)
                                    ["-DNUMBER=short" "-DINTEGER=short" "-DACCUMULATOR=short"
                                     #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"])]
        (let-release [modl (module prog)
                      short-accessor (->ShortPointerAccessor ctx hstream
                                                             (fn [^long size]
                                                               (cuda-malloc size :short))
                                                             cuda-free!)]
          (->CUFactory modl hstream short-accessor native-short
                       (->IntegerVectorEngine modl hstream) (->IntegerGEEngine modl hstream) nil nil)))))

  (defn cublas-byte [native-byte ctx hstream]
    (in-context ctx
      (with-release [prog (compile! (program integer-src)
                                    ["-DNUMBER=char" "-DINTEGER=char" "-DACCUMULATOR=char"
                                     #_"-use_fast_math" "-default-device"
                                     "-lineinfo" "-restrict"])]
        (let-release [modl (module prog)
                      byte-accessor (->BytePointerAccessor ctx hstream
                                                           (fn [^long size]
                                                             (cuda-malloc size :byte))
                                                           cuda-free!)]
          (->CUFactory modl hstream byte-accessor native-byte
                       (->IntegerVectorEngine modl hstream) (->IntegerGEEngine modl hstream) nil nil))))))
