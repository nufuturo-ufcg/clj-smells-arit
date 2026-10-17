(ns ^{:author "Dragan Djuric"}
    uncomplicate.neanderthal.internal.cpp.blas
  (:require [uncomplicate.commons
             [core :refer [with-release let-release Info info Releaseable release]]
             [utils :refer [dragan-says-ex]]]
            [uncomplicate.neanderthal
             [block :refer [buffer offset stride contiguous?]]
             [core :refer [dim entry mrows ncols symmetric? trans]]
             [math :refer [f= sqr sqrt]]]
            [uncomplicate.neanderthal.internal
             [constants :refer :all]
             [api :refer [iamax engine navigator storage region mm scal]]
             [common :refer [check-eq-navigators skip real-accessor]]
             [navigation :refer [full-storage accu-layout diag-unit? uplo=]]]
            [uncomplicate.neanderthal.internal.cpp.common :refer :all]))
