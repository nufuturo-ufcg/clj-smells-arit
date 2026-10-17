(ns riemann.transport.websockets
  "Accepts messages from external sources. Associated with a core. Sends
  incoming events to the core's streams, queries the core's index for states."
  (:require [riemann.query            :as query]
            [riemann.index            :as index]
            [riemann.pubsub           :as p]
            [riemann.test             :as test]
            [cheshire.core            :as json]
            [interval-metrics.core    :as metrics]
            [org.httpkit.server       :as http]
            [riemann.common           :refer [event-to-json ensure-event-time]]
            [riemann.core             :refer [stream!]]
            [riemann.instrumentation  :refer [Instrumented]]
            [riemann.service          :refer [Service ServiceEquiv]]
            [riemann.time             :refer [unix-time]]
            [interval-metrics.measure :refer [measure-latency]]
            [clojure.java.io          :refer [reader]]
            [clojure.tools.logging    :refer [info warn debug]]
            [clj-http.util            :refer [url-decode]]
            [clojure.string           :refer [split]]))

(defn http-query-map
  "Converts a URL query string into a map."
  [string]
  (apply hash-map
         (map url-decode
              (mapcat (fn [kv] (split kv #"=" 2))
                      (split string #"&")))))

(defn ws-pubsub-handler
  "Subscribes to a pubsub channel and streams events back over the WS conn."
  [core stats ch hs actions]
  (if-not (http/websocket? ch)
    (warn "Ignoring non-websocket request to websocket server.")
    (let [topic  (url-decode (last (split (:uri hs) #"/" 3)))
          params (http-query-map (:query-string hs))
          query  (params "query")
          pred   (query/fun (query/ast query))
          ; Subscribe persistently.
          sub    (p/subscribe! (:pubsub core) topic
                               (fn emit [event]
                                 (when (pred event)
                                   ;; http-kit's send does not provide a way
                                   ;; to measure output latency
                                   (measure-latency
                                    (:out stats)
                                    (http/send! ch (event-to-json event)))))
                               true)]
      (info "New websocket subscription to" topic ":" query)

      ; When the channel closes, unsubscribe.
      (swap! actions conj
             (fn []
               (info "Closing websocket " (:remote-addr hs) topic query)
               (p/unsubscribe! (:pubsub core) sub)))

      ; Close channel on nil msg
      (http/on-receive ch (fn [data] (when-not data (http/close ch)))))))

(defn ws-index-handler
  "Queries the index for events and streams them to the client. If subscribe is
  true, also initiates a pubsub subscription to the index topic with that
  query."
  [core stats ch hs actions]
  (if-not (http/websocket? ch)
    (warn "Ignoring non-websocket request to websocket server.")
    (let [params (http-query-map (:query-string hs))
          query  (params "query")
          ast    (query/ast query)]
      (when-let [i (:index core)]
        (doseq [event (index/search i ast)]
          (http/send! ch (event-to-json event))))
      (if (= (params "subscribe") "true")
        (ws-pubsub-handler core stats ch (assoc hs :uri "/pubsub/index") actions)
        (http/close ch)))))

(defn input-event!
  [core stats event]
  (measure-latency
   (:in stats)
   (try
     (let [event (ensure-event-time event)]
       (stream! core event)
       {})
     (catch Exception ^Exception e
            {:error (.getMessage e)}))))

(defn put-events-handler
  "Accepts events from the body of an HTTP request as JSON objects, one per
  line, and applies them to streams."
  [core stats ch req]

  (if (http/websocket? ch)
    (http/on-receive
     ch
     (fn [data]
       (if data
         (http/send! ch
                     {:status 200
                      :headers {"Content-Type" "application/json-stream"}
                      :body (json/generate-string
                             (input-event! core stats
                                           (json/parse-string data true)))})
         (http/close ch))))
    (let [body (reader (:body req))]
      (loop [line (.readLine body)]
        (when line
          (try
            (let [event (json/parse-string line true)]
              (http/send! ch
                          {:status 200
                           :headers {"Content-Type" "application/json"}
                           :body (str
                                  (json/generate-string
                                   (input-event! core stats event))
                                  "\n")}
                          false))
            (catch Exception ^Exception e
                   (http/send! ch
                               {:status 200
                                :headers {"Content-Type" "application/json"}
                                :body (str
                                       (json/generate-string
                                        {:error (.getMessage e)}) "\n")}
                               true)))
          (recur (.readLine body))))
      (http/close ch))))

(defn ws-handler
  "Returns a function which is called with new websocket connections.
  Responsible for routing requests to the appropriate handler."
  [core stats]
  (fn handle [req]
    (http/with-channel req ch
      (try
        (let [actions (atom [])]
          (debug "Websocket connection from" (:remote-addr req)
                 (:uri req)
                 (:query-string req))

                                        ; Stats
          (when (http/websocket? ch)
            (swap! (:conns stats) inc)
            (http/on-close ch
                           (fn [_]
                             (swap! (:conns stats) dec)
                             (doseq [action @actions]
                               (action)))))

          ;; Route request
          (condp re-matches (:uri req)
            #"/events/?"       (put-events-handler @core stats ch req)
            #"/index/?"        (ws-index-handler @core stats ch req actions)
            #"/pubsub/[^/]+/?" (ws-pubsub-handler @core stats ch req actions)
            (do
              (info "Unknown URI " (:uri req) ", closing")
              (http/close ch))))

        (catch Exception t
          (do
            (warn t "ws-handler caught; closing websocket connection.")
            (http/close ch)))))))
