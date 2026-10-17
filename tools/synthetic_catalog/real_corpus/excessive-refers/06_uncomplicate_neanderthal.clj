(ns ^{:author "Dragan Djuric"}
    uncomplicate.neanderthal.internal.cpp.lapack
  (:refer-clojure :exclude [abs])
  (:require [uncomplicate.commons
             [core :refer [with-release let-release info]]
             [utils :refer [dragan-says-ex cond-into]]]
            [uncomplicate.clojure-cpp
             :refer [long-ptr int-ptr byte-pointer int-pointer long-pointer pointer get-entry]]
            [uncomplicate.neanderthal
             [core :refer [dim mrows ncols col trans symmetric? entry!]]
             [block :refer [stride row? contiguous?]]
             [math :refer [f=]]]
            [uncomplicate.neanderthal.math :refer [sqrt pow abs]]
            [uncomplicate.neanderthal.internal
             [common :refer [skip real-accessor check-stride]]
             [api :refer [factory index-factory engine data-accessor raw mm scal flip
                          create-vector create-ge create-gb create-sb create-sy fits-navigation?
                          nrm1 nrmi copy nrm2 amax trf tri trs con sv navigator region storage
                          view-ge]]
             [navigation :refer [diag-unit? lower? upper? dostripe-layout full-storage]]]
            [uncomplicate.neanderthal.internal.constants
             :refer [blas-layout blas-uplo blas-transpose blas-diag]]
            [uncomplicate.neanderthal.internal.cpp.blas
             :refer [full-storage-map band-storage-map band-storage-reduce]])
  (:import [uncomplicate.neanderthal.internal.api DiagonalMatrix LayoutNavigator Region]
           uncomplicate.neanderthal.internal.navigation.BandStorage))
