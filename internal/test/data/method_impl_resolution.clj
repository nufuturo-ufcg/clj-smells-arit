(ns method-impl-resolution
  (:import [com.google.common.cache CacheLoader]))
;; Method implementations name interface methods, not core functions, and
;; bind their parameters lexically.
(defn loader [store]
  (reify CacheLoader
    (load [_ k] (get store k))))
(defn loader2 []
  (proxy [CacheLoader] []
    (load [k] (str k))))
(deftype Box [items]
  clojure.lang.Counted
  (count [_] (count items)))
(defn real-load [] (load "other"))
(defn shadowing-param [xs]
  (reify clojure.lang.IFn
    (invoke [_ map] (vec (map inc xs)))))
