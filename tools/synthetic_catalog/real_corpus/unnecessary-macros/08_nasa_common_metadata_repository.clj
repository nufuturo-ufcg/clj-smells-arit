(ns cmr.spatial.math
  ;; This line is needed to prevent a confusion on which abs should be used as reported by clj-kondo
  (:refer-clojure :exclude [abs])
  [:require
   [primitive-math]]
  (:import
   (net.jafama StrictFastMath)))

(defmacro atan [v]
  `(StrictFastMath/atan ~v))
