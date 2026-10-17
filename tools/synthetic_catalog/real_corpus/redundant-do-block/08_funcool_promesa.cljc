(ns ^:no-doc promesa.exec.csp.channel
  (:refer-clojure :exclude [take])
  (:require
   [promesa.core :as p]
   [promesa.exec :as px]
   [promesa.exec.csp.mutable-list :as mlist]
   [promesa.protocols :as pt]
   [promesa.util :as util]))

(defn promise->handler
  {:no-doc true}
  ([p] (promise->handler p true))
  ([p blockable?]
   (let [lock    (util/mutex)
         active? (atom true)]
     (reify
       pt/ILock
       (-lock [_] (pt/-lock lock))
       (-unlock [_] (pt/-unlock lock))

       pt/IHandler
       (-active? [_]
         (and (p/pending? p)
              (deref active?)))
       (-blockable? [_] blockable?)
       (-commit [_]
         (and (compare-and-set! active? true false)
              (fn
                ([v]
                 (when (p/pending? p)
                   (p/resolve p v)))
                ([v c]
                 (when (p/pending? p)
                   (if c
                     (p/reject p c)
                     (p/resolve p v)))))))))))

(defn commit
  "A convenience helper that locks, checks and return commited handler
  callbale given an instance of handler."
  {:no-doc true}
  [handler]
  (try
    (pt/-lock handler)
    (when (pt/-active? handler)
      (pt/-commit handler))
    (finally
      (pt/-unlock handler))))

(defn- process-pending-puts
  "This is the loop that processes the pending puts after a succesfull
  take operation (that has probably have freed a slot in the buffer)."
  [this executor puts add-fn buf]
  (loop [done? false]
    (if (or (pt/-full? buf) done?)
      done?
      (if-let [[putter val] (mlist/remove-first puts)]
        (do
          (if-let [put-fn (commit putter)]
            (let [done? (reduced? (add-fn this buf val))]
              (px/exec executor (partial put-fn true))
              (recur done?))
            (recur done?)))

        done?))))

(defn take
  {:no-doc true}
  ([port]
   (let [d (p/deferred)]
     (pt/-take port (promise->handler d))
     d))
  ([port timeout-duration timeout-value]
   (let [d (p/deferred)
         h (promise->handler d)
         t (px/schedule timeout-duration
                        #(when-let [f (commit h)]
                           (f timeout-value)))]
     (pt/-take port h)
     (p/finally d (fn [_ _] (p/cancel t))))))

(defn- add-fn
  ([b] b)
  ([b itm]
   (assert (not (nil? itm)))
   (pt/-offer b itm)
   b))
