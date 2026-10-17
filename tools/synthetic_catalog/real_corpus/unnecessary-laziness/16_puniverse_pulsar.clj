(ns co.paralleluniverse.pulsar.core
  "Pulsar is an implementation of lightweight threads (fibers),
  Go-like channles and Erlang-like actors for the JVM"
(:refer-clojure :exclude [promise await bean])
(:import [java.util.concurrent TimeUnit ExecutionException TimeoutException Future]
         [co.paralleluniverse.strands Strand Stranded]
         [co.paralleluniverse.strands SuspendableCallable]
         [co.paralleluniverse.fibers DefaultFiberScheduler FiberScheduler Fiber Joinable FiberUtil]
         [co.paralleluniverse.fibers.instrument]
         [co.paralleluniverse.strands.channels Channel Channels Channels$OverflowPolicy ReceivePort SendPort
          Selectable Selector SelectAction
          TickerChannelConsumer Topic ReceivePortGroup
          IntChannel LongChannel FloatChannel DoubleChannel
          IntSendPort LongSendPort FloatSendPort DoubleSendPort
          IntReceivePort LongReceivePort FloatReceivePort DoubleReceivePort]
         [co.paralleluniverse.strands.dataflow Val Var]
         [co.paralleluniverse.pulsar ClojureHelper ChannelsHelper ClojureFiberAsync]
         ; for types:
         [clojure.lang Keyword Sequential IObj IMeta IDeref ISeq IPersistentCollection IPersistentVector IPersistentMap])
(:require [co.paralleluniverse.pulsar.interop :refer :all]))

(defmacro ^{:private true} assert-args
  [& pairs]
  `(do (when-not ~(first pairs)
         (throw (IllegalArgumentException.
                 (str (first ~'&form) " requires " ~(second pairs) " in " ~'*ns* ":" (:line (meta ~'&form))))))
     ~(let [more (nnext pairs)]
        (when more
          (list* `assert-args more)))))

(defn- ops-args
  "Used to simplify optional parameters in functions.
  Takes a sequence of [predicate? default] pairs, and a sequence of arguments. Tests the first predicate against
  the first argument. If the predicate succeeds, emits the argument's value; if not - the default, and tries the
  next pair with the argument. Any remaining arguments are copied to the output as-is."
  {:no-doc true}
  [pds xs]
  (if (seq pds)
    (let [[p? d] (first pds)
          x      (first xs)]
      (if (p? x)
        (cons x (ops-args (rest pds) (rest xs)))
        (cons d (ops-args (rest pds) xs))))
    (seq xs)))

(defn convert-duration
  "Converts a time duration from one time unit to another.
  x is the duration; `from-unit` and `to-unit` are the source
  and target units repsectively, given as either a j.u.c.TimeUnit instance
  or as a keyword, as specified by `->timeunit`."
  [x from-unit to-unit]
  (.convert (->timeunit to-unit) x (->timeunit from-unit)))

(defn unwrap-exception*
  {:no-doc true}
  [^Throwable e]
  (if
    (or (instance? ExecutionException e)
        (and (= (.getClass e) RuntimeException) (.getCause e)))
    (unwrap-exception* (.getCause e))
    e))

(defmacro unwrap-exception
  {:no-doc true}
  [& body]
  `(try
     ~@body
     (catch Exception e#
       (throw (unwrap-exception* e#)))))

(defn suspendable!
  "Makes a function suspendable."
  ([f]
   (when f
     (ClojureHelper/retransform f nil)))
  ([x prot]
   (ClojureHelper/retransform x prot)))

(defmacro sfn
  "Creates a suspendable function that can be used by a fiber or actor.
  Used exactly like `fn`"
  [& expr]
  `(suspendable! (fn ~@expr)))

(defmacro defsfn
  "Defines a suspendable function that can be used by a fiber or actor.
  Used exactly like `defn`"
  [& expr]
  `(do
     (defn ~@expr)
     (suspendable! ~(first expr))))

(defmacro sreify
  "Creates a suspendable implementation of a protocol or interface.
  sreify is to reify what sfn is to fn."
  [& expr]
  `(suspendable! (reify ~@expr)
                 ~(vec (map (fn [x] (if-let [iface (:on-interface x)] iface x))
                            (map eval (filter symbol? expr))))))

(defsfn kps-args
        {:no-doc true}
        [args]
        (let [aps (partition-all 2 args)
              [opts-and-vals ps] (split-with #(keyword? (first %)) aps)
              options (into {} (map vec opts-and-vals))
              positionals (reduce into [] ps)]
          [options positionals]))

(defn current-fiber
  "Returns the currently running lightweight-thread or `nil` if none."
  []
  (Fiber/currentFiber))

(defn- current-scheduler []
  (when-let [^Fiber f (current-fiber)]
    (.getScheduler f)))

(def ^FiberScheduler default-fiber-scheduler
  "A global fiber scheduler. The scheduler uses all available processor cores."
  (DefaultFiberScheduler/getInstance))

(defn ^FiberScheduler get-scheduler
  {:no-doc true}
  [^FiberScheduler scheduler]
  (or scheduler (current-scheduler) default-fiber-scheduler))

(defn ^Fiber create-fiber
  "Creates, but does not start a new fiber (a lightweight thread) running in a fork/join pool.

  It is much preferable to use `spawn-fiber`."
  [& args]
  (let [[^String name ^FiberScheduler scheduler ^Integer stacksize f] (ops-args [[string? nil] [#(instance? FiberScheduler %) default-fiber-scheduler] [integer? -1]] args)]
    (Fiber. name (get-scheduler scheduler) (int stacksize) (->suspendable-callable f))))

(defn start
  "Starts a fiber created with `create-fiber`."
  [^Fiber fiber]
  (.start fiber))

(defmacro spawn-fiber
  "Creates and starts a new fiber.

  f - the function to run in the fiber.
  args - (optional) arguments for the function

  Options:
  :name str     - the fiber's name
  :stack-size n - the fiber's initial stack size
  :scheduler    - the fiber schdeuler in which the fiber will run
  "
  {:arglists '([:name? :stack-size? :scheduler? f & args])}
  [& args]
  (let [[{:keys [^String name ^Integer stack-size ^FiberScheduler scheduler] :or {stack-size -1}} body] (kps-args args)]
    `(let [f#     (suspendable! ~(if (== (count body) 1) (first body) `(fn [] (apply (suspendable! (first ~body)) (rest ~body)))))
           fiber# (co.paralleluniverse.fibers.Fiber. ~name (get-scheduler ~scheduler) (int ~stack-size) (->suspendable-callable f#))]
       (.start fiber#))))

(defmacro fiber
  "Runs the given body in a newly created fiber and returns it."
  [& body]
  `(spawn-fiber (fn [] ~@body)))

(defsfn ^:private join*
  ([s]
   (unwrap-exception
     (cond
       (instance? Joinable s) (.get ^Joinable s)
       (instance? Strand s)   (Strand/join s)
       (instance? Thread s)   (Strand/join (Strand/of ^Thread s))
       (instance? co.paralleluniverse.actors.ActorRef s) (co.paralleluniverse.actors.LocalActor/get s)
       :else (throw (IllegalArgumentException. (str "Cannot join " s))))))
  ([timeout unit s]
   (unwrap-exception
     (cond
       (instance? Joinable s) (.get ^Joinable s timeout (->timeunit unit))
       (instance? Strand s)   (Strand/join s timeout (->timeunit unit))
       (instance? Thread s)   (Strand/join (Strand/of ^Thread s) timeout (->timeunit unit))
       (instance? co.paralleluniverse.actors.ActorRef s) (co.paralleluniverse.actors.LocalActor/get s timeout (->timeunit unit))
       :else (throw (IllegalArgumentException. (str "Cannot join " s)))))))

(defsfn join
  "Awaits the termination of the given strand or strands, and returns
  their result, if applicable.

  If a single strand is given, its result is returned;
  if a collection - then a collection of the respective results.

  Note that for threads, the result is always `nil`, as threads don't return a value.

  If a timeout is supplied and it elapses before the strand has terminated,
  a j.u.c.TimeoutException is thrown.

  s       - either a strand or a collection of strands.
  timeout - how long to wait for the strands termination
  unit    - the unit of the timeout duration. TimeUnit or keyword as in `->timeunit`"
  ([s]
   (if (coll? s)
     (doall (map join* s))
     (join* s)))
  ([timeout unit s]
   (if (coll? s)
     (loop [nanos (long (convert-duration timeout unit :nanos))
            res []
            ss s]
       (when (not (pos? nanos))
         (throw (TimeoutException.)))
       (if (seq? ss)
         (let [start (long (System/nanoTime))
               r (join* (first ss) nanos TimeUnit/NANOSECONDS)]
           (recur (- nanos (- (System/nanoTime) start))
                  (conj res r)
                  (rest ss)))
         (seq res)))
     (join* timeout unit s))))
