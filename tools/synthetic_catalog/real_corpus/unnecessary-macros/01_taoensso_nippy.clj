(ns ^:no-doc taoensso.nippy.impl
  "Private misc utils, don't use."
  (:require
   [clojure.string  :as str]
   [taoensso.truss  :as truss]
   [taoensso.encore :as enc]))

(defmacro md-count?  [n] `(<= ~n Short/MAX_VALUE))
