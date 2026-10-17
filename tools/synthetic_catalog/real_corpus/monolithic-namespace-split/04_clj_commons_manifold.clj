(ns manifold.time
  {:author "Zach Tellman"
   :doc    "This namespace contains methods for converting units of time, with milliseconds as the base representation, and for deferring execution of functions to some time in the future.  In practice, the methods here are not necessary to use Manifold effectively - `manifold.deferred/timeout` and `manifold.stream/periodically` are more directly useful - but they are available for anyone who should need them."}
  (:require
    [clojure.tools.logging :as log]
    [manifold.executor :as ex]
    [clojure.string :as str]
    [manifold.utils :refer [definterface+]]
    [potemkin.types :refer [defprotocol+]]
    [clj-commons.primitive-math :as p])
  (:import
    [java.util
     Calendar
     TimeZone]
    [java.util.concurrent
     Future
     Executor
     Executors
     TimeUnit
     ScheduledExecutorService
     ScheduledThreadPoolExecutor
     TimeoutException]))

(in-ns 'manifold.time)
