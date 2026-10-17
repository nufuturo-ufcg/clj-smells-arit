(ns methodical.impl.standard
  (:require
   [clojure.core.protocols :as clojure.protocols]
   [clojure.datafy :as datafy]
   [methodical.interface :as i]
   [methodical.util.describe :as describe]
   [pretty.core :as pretty])
  (:import
   (clojure.lang Named)
   (methodical.interface Dispatcher MethodCombination MethodTable MultiFnImpl)))

(defn- maybe-name [^MultiFnImpl impl]
  (if-let [nm (and (instance? Named impl) (name impl))]
    (str " " nm)
    ""))

(defn- handle-effective-method-exception [^Exception e mta]
  (if-let [dispatch-val (::unmatched-dispatch-value (ex-data e))]
    (throw (UnsupportedOperationException.
             (format "No matching%s method for dispatch value %s" (if-let [nm (:name mta)]
                                                                    (str " " nm)
                                                                    "")
                                                                  (pr-str dispatch-val))))
    ;; this wasn't an :unmatched-dispatch-value situation; just rethrow it
    (throw e)))

(defn- ^:static effective-method [^MultiFnImpl impl dispatch-value]
  (or (.effective-method impl dispatch-value)
      (throw
       (ex-info (format "No matching%s method for dispatch value %s" (maybe-name impl) (pr-str dispatch-value))
                {::unmatched-dispatch-value dispatch-value}))))

(defmacro ^:private invoke-multi
  "Utility macro for finding the effective method of `impl`, given the `args`, then catching an Exception on invoking
  the `effective-method`, where we look for the special case of `::unmatched-dispatch-value`. If we find that, we
  rethrow a regular `UnsupportedOperationException` including the method name and `pr-str` of the  unmatched dispatch
  value. If not, we simply rethrow the exception since it's not ours to handle."
  [impl mta & args]
  `(try
     (let [em# (effective-method ~impl (.dispatch-value ^Dispatcher (.dispatcher ~impl) ~@args))]
       (em# ~@args))
     (catch Exception e#
       (handle-effective-method-exception e# ~mta))))
