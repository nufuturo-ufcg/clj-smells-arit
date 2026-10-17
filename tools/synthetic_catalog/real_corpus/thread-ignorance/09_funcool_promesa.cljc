(ns promesa.util
  (:refer-clojure :exclude [with-open])
  (:require [promesa.protocols :as pt])
  #?(:clj
     (:import
      java.lang.reflect.Method
      java.util.concurrent.CancellationException
      java.util.concurrent.CompletionException
      java.util.concurrent.CompletionStage
      java.util.concurrent.ExecutionException
      java.util.concurrent.TimeUnit
      java.util.concurrent.TimeoutException
      java.util.concurrent.locks.ReentrantLock)))

(defn close
  ([o]
   (pt/-close o))
  ([o reason]
   (pt/-close o reason)))

(defmacro with-open
  [bindings & body]
  {:pre [(vector? bindings)
         (even? (count bindings))
         (pos? (count bindings))]}

  (when (:ns &env)
    (throw (ex-info "cljs not supported on with-dispatch! macro" {})))

  (reduce (fn [acc bindings]
            `(let ~(vec bindings)
               (try
                 ~acc
                 (finally
                   (let [target# ~(first bindings)]
                     (if (instance? java.lang.AutoCloseable target#)
                       (.close ^java.lang.AutoCloseable target#)
                       (pt/-close target#)))))))
          `(do ~@body)
          (reverse (partition 2 bindings))))
