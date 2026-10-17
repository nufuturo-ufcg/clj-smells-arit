(ns riemann.core
  "Binds together an index, servers, and streams."
  (:require [riemann.time :refer [unix-time]]
            [riemann.common :refer [deprecated localhost event]]
            riemann.streams
            [riemann.service :as service :refer [Service ServiceEquiv]]
            [riemann.index :as index :refer [Index]]
            [riemann.pubsub :as ps]
            [riemann.test :as test]
            [riemann.instrumentation :as instrumentation :refer [Instrumented]]
            [clojure.tools.logging :refer [warn info]]
            clojure.set))

(defn stream!
  "Applies an event to the streams in this core."
  [core event]
  (instrumentation/measure-latency (:streaming-metric core)
    (doseq [stream (:streams core)]
      (stream event))))

(defn core-services
  "All services in a core--both the :services list and named services like the
  index."
  [core]
  (remove nil?
          (concat [(:index core)
                   (:pubsub core)]
                (:services core))))

(defn instrumentation-service
  "Returns a service which samples instrumented services in its core every
  interval seconds, and sends their events to the core itself."
  [opts]
  (let [interval (long (* 1000 (get opts :interval 10)))
        enabled? (if-not test/*testing*
                   (get opts :enabled? true)
                   false)]
    (service/thread-service
      ::instrumentation [interval enabled?]
      (fn measure [core]
        (Thread/sleep interval)
        (try
          ; Take events from core and instrumented services
          (let [base (event {:host (localhost)
                             ; Default TTL of 2 intervals, and convert ms to s.
                             :ttl  (long (/ interval 500))})
                events (mapcat instrumentation/events
                            (concat
                              [core
                               ; lol circular deps
                               (deref (find-var 'riemann.transport/instrumentation))]
                              (filter instrumentation/instrumented?
                                      (core-services core))))]

            (if enabled?
              ; Stream each event through this core
              (doseq [event events]
                (stream! core (merge base event)))
              ; Ensure we consume all events, to avoid overflowing stats
              (dorun events)))

          (catch Exception e
            (warn e "instrumentation service caught")))))))

(defrecord Core
  [streams services index pubsub streaming-metric]

  Instrumented
  (events [this]
          (instrumentation/events streaming-metric)))

(defn core
  "Create a new core."
  []
  (Core. []
         [(instrumentation-service {})]
         nil
         (ps/pubsub-registry)
         (instrumentation/rate+latency {:service "streams"
                                        :tags ["riemann"]})))

(defn conj-service
  "Adds a service to a core. Throws if any existing services would conflict. If
  force? is passed, dissoc's any conflicting services."
  ([core service] (conj-service core service false))
  ([core service force?]
   (if force?
     ; Remove conflicts and conj service
     (assoc core :services
            (conj (remove #(service/conflict? service %)
                          (:services core))
                  service))

     ; Throw if conflicts arise
     (let [conflicts (filter #(service/conflict? service %)
                             (core-services core))]
       (when-not (empty? conflicts)
         (throw (IllegalArgumentException.
                  (binding [*print-level* 3]
                    (str "won't conj service: " (pr-str service)
                         " would conflict with " (pr-str conflicts))))))
       (update-in core [:services] conj service)))))
