(ns mcp.client
  "functions that use mcp definitions in prompts-files to
   to create function definitions"
  (:require
   [babashka.fs :as fs]
   [cheshire.core :as json]
   [clojure.core.async :as async]
   [clojure.edn :as edn]
   [clojure.string :as string]
   [docker]
   [flatland.ordered.map :refer [ordered-map]]
   interpolate
   [jsonrpc.logger :as logger]
   [prompts.core :refer [get-prompts-dir]]
   repl))

(def counter (atom 0))

(defn- mcp-stdio-stateless-server
  "create a running container with an attached socket-client 
   and two running Threads 
     - one that is reading messages from the socket (both stdout and stderr)
     - one that is waiting for the engine to indicate that the container has stopped
   plus a go loop
     - watches socket read-loop channel for jsonrpc messages
     - implements timeouts of 15s for at least something to happen on the channel
     - if the read-loop channel closes, we will write send a close message explicitly
     - request jsonrpc messages create promises that can only resolve with matching responses
  
   returns 
     a map with the container info plus request and notification functions 
     to write jsonrpc messages to the socket,
     and a dead-channel channel that will emit :stopped or :closed when the container stops for any reason 
       the request function will return a promise channel for the matching response
       the notification function is just a side-effect
  "
  [container]
  (docker/check-then-pull container)
  (let [x (docker/create (assoc container
                                :opts {:StdinOnce true
                                       :OpenStdin true
                                       :AttachStdin true}))
        response-promises (atom {})
        socket-channel (docker/attach-socket (:Id x))
        c (async/chan 10)
        dead-channel (async/chan)]
    (docker/start x)

    ;; process the output-stream of the container
    ;; should only end once the stream closes
    (.start ^Thread
     (Thread.
      (fn []
        (docker/read-loop socket-channel c))))
    (.start ^Thread
     (Thread.
      (fn []
        (docker/wait x)
        (async/put! c :stopped))))

    ;; processs the jsonrpc messages
    (async/go-loop []
      (let [block (async/alt!
                    c ([v _] v)
                    (async/timeout 15000) :timeout)]
        (cond
          ;; read-loop has stopped or timed out
          (#{:stopped :timeout} block)
          (do
            (logger/info "stopped/timeout")
            (async/put! dead-channel block)
            block)

          ;; real stdout message
          (and block (:stdout block))
          (let [messages (->> (string/split (:stdout block) #"\n")
                              (map (fn [s] (try (json/parse-string s keyword) (catch Throwable _ s))))
                              (into []))]
            (doseq [message messages]
              (if-let [p (get @response-promises (:id message))]
                (async/put! p message)
                (logger/debug (:image container) message)))
            (recur))

          ;; channel is closed
          (nil? block)
          (async/put! dead-channel :closed) 

          ;; non-stdout message - show to user
          :else
          (recur))))

       ;; add a function to send a jsonrpc request
    (assoc x
           :request
           (fn [message]
             (let [id (swap! counter inc)
                   c (async/promise-chan)]
               (swap! response-promises assoc id c)
               (try
                 (docker/write-to-stdin socket-channel (str (json/generate-string (assoc message :id id :jsonrpc "2.0")) "\n\n"))
                 (catch Throwable t
                   (logger/error (format "error writing to container stdin: %s" t))))
               c))
           :notification
           (fn [message]
             (try
               (docker/write-to-stdin socket-channel (str (json/generate-string (assoc message :jsonrpc "2.0")) "\n\n"))
               (catch Throwable t
                 (logger/error (format "error writing to container stdin: %s" t)))))
           :dead-channel dead-channel
           :remove (fn []
                     (docker/kill-container x)
                     (docker/delete x)))))

(defn with-running-mcp
  "send a message to an mcp server and then shuts it down
  params
  container-definition - for the mcp server
  f - function to generate a jsonrpc request to send post initialize
  f1 - handler for the jsonrpc response (could be just identity)
  returns
  a channel with the response the channel will emit [] if there's an error"
  [{:keys [_server-id stateful] :as container-definition} f]
  (try
    (let [{:keys [request notification dead-channel remove] :as server} (mcp-stdio-stateless-server container-definition)]
      ;; dead-channel will emit either :stopped :timeout or :closed
      (async/go
        (try
          (let [{:keys [initialized response]}
                (async/alt!
                  (request {:method "initialize"
                            :params {:protocolVersion "2024-11-05"
                                     :capabilities {:tools {}
                                                    :resources {}}
                                     :clientInfo {:name "docker"
                                                  :version "0.1.0"}}}) ([v _] {:initialized true
                                                                               :response v})
                  dead-channel ([v _] v)
                  (async/timeout 15000) :timeout)]
            (if (true? initialized)
              (do
                (notification {:method "notifications/initialized" :params {}})
                (async/<! (f server response)))
              (do
                (logger/error (format
                               "mcp server channel did not initialize for %s"
                               (:image container-definition)))
                {:error :did-not-initialize})))
          (finally
            (when (and
                   (not (true? stateful))
                   (not (= "false" (System/getenv "GATEWAY_CONTAINER_RM"))))
              (remove))))))
    (catch Throwable t
      (logger/error (.getMessage t))
      (when (ex-data t)
        (logger/error (ex-data t)))
      (let [c (async/promise-chan)]
        (async/>!! c {:error :exception :ex t})
        c))))

(defn- send-get-resource-message [params {:keys [request dead-channel]} _]
  (async/go
    (let [response (async/alt!
                     (request {:method "resources/read" :params params}) ([v _] v)
                     dead-channel ([v _] v)
                     (async/timeout 15000) :timeout)]
      (logger/info "resource: " (-> response :result :contents))
      response)))

(defn -get-resource
  "list resources"
  [container-definition params]
  (async/<!!
   (with-running-mcp
     (-> container-definition
         ;; interpolate parameters 
         ;;   - this looks like an extra step because it's not running from the tools module
         ((fn [c] (interpolate/container-definition
                   {:container (dissoc c :parameter-values)}
                   {}
                   (json/generate-string
                    (:parameter-values c)))))
         ;; inject secrets from container definition
         docker/inject-secret-transform)
     (partial send-get-resource-message params))))

(defn get-function-factory
  "construct a function that gets a resource from a container MCP
     return a channel to emit matching resources. The channel might just close if nothing is found."
  [container-definition]
  (fn get [uri]
    (let [c (async/chan)]
      ; TODO skip this if this container-definition doesn't support this uri
      (let [response (-get-resource container-definition {:uri uri})]
        (when (:error response)
          (logger/error (format "%s error getting resource %s" container-definition response)))
        (when (:result response)
          (logger/info (format "%s got resource %s" (:image container-definition) (:result response)))
          (async/put! c (:result response))))
      (async/close! c)
      c)))
