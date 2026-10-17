(ns
    ^{:doc "A simple event system that handles both synchronous and asynchronous events."
      :author "Jeff Rose & Sam Aaron"}
    overtone.libs.handlers
  (:import (java.util.concurrent Executors ExecutorService))
  (:use [clojure.stacktrace]))

(def ^{:dynamic true} *FORCE-SYNC?* false)

(def ^{:dynamic true} *log-fn* nil)

(defn- swap-returning-prev!
  "Similar to swap! except returns vector containing the previous and new values

  (def a (atom 0))
  (swap-returning-prev! a inc) ;=> [0 1]"
  [atom f & args]
  (loop []
    (let [old-val  @atom
          new-val  (apply f (cons old-val args))
          success? (compare-and-set! atom old-val new-val)]
      (if success?
        [old-val new-val]
        (recur)))))

(defn- log-error
  "Log error using *log-fn* if bound."
  [& msgs]
  (when *log-fn*
    (*log-fn* (apply str msgs))))

(defn- mk-emh
  "Create a new event matcher handler map containing only the
  necessary keys."
  [syncs asyncs sync-one-shots async-one-shots]
  (let [emh {}
        emh (if syncs (assoc emh :syncs syncs) emh)
        emh (if asyncs (assoc emh :asyncs asyncs) emh)
        emh (if sync-one-shots (assoc emh :sync-one-shots sync-one-shots) emh)
        emh (if async-one-shots (assoc emh :async-one-shots async-one-shots) emh)]
    emh))

(defn- emh-keys
  "Return a seq of keys for the given event matcher handlers"
  [emh]
  (concat
   (keys (:syncs emh))
   (keys (:asyncs emh))
   (keys (:sync-one-shots emh))
   (keys (:async-one-shots emh))))

(defn- emh-count-handlers
  "Count the handlers associated with a single event matcher"
  [emh]
  (count (emh-keys emh)))

(defn- emh-rm-key
  "Returns a new event matcher handlers map with the key
  removed. Returns nil if no matchers remain."
  [emh key]
  (let [syncs           (dissoc (:syncs emh) key)
        asyncs          (dissoc (:asyncs emh) key)
        sync-one-shots  (dissoc (:sync-one-shots emh) key)
        async-one-shots (dissoc (:async-one-shots emh) key)
        new-emh         (mk-emh syncs asyncs sync-one-shots async-one-shots)]
    (when-not (= 0 (emh-count-handlers new-emh))
      new-emh)))

(defn- emh-rm-specific-handler
  "Removes specific handler fun from emh.
  Returns a new event matcher handlers map with the key
  removed. Returns nil if no matchers remain."
  [emh key handler]
  (let [syncs           (if (= handler (get-in emh [:syncs key]))
                          (dissoc (:syncs emh) key)
                          (:syncs emh))
        asyncs          (if (= handler (get-in emh [:asyncs key]))
                          (dissoc (:asyncs emh) key)
                          (:asyncs emh))
        sync-one-shots  (if (= handler (get-in emh [:sync-one-shots key]))
                          (dissoc (:sync-one-shots emh) key)
                          (:sync-one-shots emh))
        async-one-shots (if (= handler (get-in emh [:async-one-shots key]))
                          (dissoc (:async-one-shots emh) key)
                          (:async-one-shots emh))
        new-emh         (mk-emh syncs asyncs sync-one-shots async-one-shots)]
    (when-not (= 0 (emh-count-handlers new-emh))
      new-emh)))

(defn- handlers-rm-key
  "Returns a new handlers map ommitting key. Removes event-matcher if
  no handlers remain."
  [handlers key]
  (into {} (filter second
                   (map (fn [[event-matcher emh]]
                          [event-matcher (emh-rm-key emh key)])
                        handlers))))

(defn- run-handler
  "Apply the handler to the args - handling exceptions
  gracefully. Also remove the handler if it
  returns :overtone/remove-handler."
  [key handler event-map hp]
  (try
    (let [res (handler event-map)]
      (when (= :overtone/remove-handler res)
        (remove-handler! hp key))
      res)
    (catch Exception e
      (log-error (str "Handler Exception - with event-map: ") event-map "\n"
                 "Make sure that your callback function accepts at least 1 argument \n"
                 "A function signature containing #(fn) will not work without % in it.\n"
                 "Use (fn [event-map] (fn)) instead.\n"
                 (with-out-str (.printStackTrace e))))))

(defn- run-handlers
  "Trigger all handlers in keyed-fns."
  [keyed-fns event-map hp]
  (doseq [[k f] keyed-fns]
    (run-handler k f event-map hp)))

(defn- handlers-rm-specific-handler
  "Returns a new handlers map ommitting handler. Removes event-matcher
  if no handlers remain."
  [handlers key handler]
  (into {} (filter second
                   (map (fn [[event-matcher emh]]
                          [event-matcher (emh-rm-specific-handler emh key handler)])
                        handlers))))

(defn- remove-specific-handler!
  "Remove a specific handler fn. Returns true if the fn was removed."
  [hp key handler]
  (let [[o n] (swap-returning-prev! (:handlers hp)
                                    (fn [handlers]
                                      (handlers-rm-specific-handler handlers key handler)))]
    (not= o n)))

(defn- default-matcher-fn
  "Default matcher fn which simply returns the first matcher in
  event-matchers which is identical to event-matcher."
  [event-matchers event-matcher]
  (let [match (some #{event-matcher} event-matchers)]
    [match]))

(defn- handlers-matching-emhs
  "Returns a seq of event-matcher handler-maps. Currently just returns
  a direct match of the event-matcher, but this could be expanded to
  more interesting match algorithms. Therefore returns a result seq to
  allow these future algorithms to return more than one match."
  [handlers event-matcher matcher-fn]
  (let [matchers (matcher-fn (keys handlers) event-matcher)
        matchers (or matchers [])]
    (map (fn [matcher] (get handlers matcher)) matchers)))

(defn- run-one-shot-handlers
  "Trigger all handlers in keyed-fns, removing them from the hp before
  doing so (to ensure they're only executed once)."
  [keyed-fns event-map hp]
  (doseq [[k handler] keyed-fns]
    (when (remove-specific-handler! hp k handler)
      (run-handler k handler event-map hp))))

(defn- emhs-handle-async-one-shots
  [emhs event-map ^ExecutorService pool hp]
  (let [keyed-fns (apply merge (map (fn [emh] (:async-one-shots emh)) emhs))]
    (if *FORCE-SYNC?*
      (run-one-shot-handlers keyed-fns event-map hp)
      (.execute pool #(run-one-shot-handlers keyed-fns event-map hp)))))

(defn- emhs-handle-sync-one-shots
  [emhs event-map pool hp]
  (let [keyed-fns (apply merge (map (fn [emh]  (:sync-one-shots emh)) emhs))]
    (run-one-shot-handlers keyed-fns event-map hp)))

(defn- emhs-handle-async-events
  "Runs all async handlers in a thread pool. If binding *FORCE-SYNC?*
  is true, forces async handlers to execute synchronously on the
  current thread."
  [emhs event-map ^ExecutorService pool hp]
  (let [keyed-fns (apply merge (map (fn [emh] (:asyncs emh)) emhs))]
    (if *FORCE-SYNC?*
      (run-handlers keyed-fns event-map hp)
      (.execute pool #(run-handlers keyed-fns event-map hp)))))

(defn- emhs-handle-sync-events
  "Runs all sync handlers on the current thread."
  [emhs event-map pool hp]
  (let [keyed-fns (apply merge (map (fn [emh]  (:syncs emh)) emhs))]
    (run-handlers keyed-fns event-map hp)))

(defn- hp-matching-emhs
  "Returns a seq of matching event method handles from the handle-pool
  matching event-matcher."
  [hp event-matcher matcher-fn]
  (let [handlers @(:handlers hp)]
    (handlers-matching-emhs handlers event-matcher matcher-fn)))

(defn remove-handler!
  "Remove event handler in handler pool with key."
  [hp key]
  (swap! (:handlers hp)
         (fn [handlers]
           (handlers-rm-key handlers key)))
  :handler-removed)

(defn event
  "Create a new event for handlers matching event-matcher. This will
  trigger all matching handlers and call them with event-info as an
  argument.

  Accepts an optional matcher-fn which can override the default
  behaviour for matching the event's event-matcher with the registered
  handlers' event-matchers. This fn should take two arguments - a seq
  of the registered event-matchers and the event-matcher of the
  event. It must return a list of matching event-matchers."
  ([hp event-matcher] (event hp event-matcher {} default-matcher-fn))
  ([hp event-matcher event-info] (event hp event-matcher event-info default-matcher-fn))
  ([hp event-matcher event-info matcher-fn]
   (let [pool (:pool hp)
         emhs (hp-matching-emhs hp event-matcher matcher-fn)]
     (emhs-handle-async-events emhs event-info pool hp)
     (emhs-handle-sync-events emhs event-info pool hp)
     (emhs-handle-async-one-shots emhs event-info pool hp)
     (emhs-handle-sync-one-shots emhs event-info pool hp))))
