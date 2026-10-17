(ns clojure-mcp.nrepl
  (:require
   [clojure.java.io :as io]
   [clojure.string :as str]
   [nrepl.core :as nrepl]
   [nrepl.misc :as nrepl.misc]
   [nrepl.transport :as transport]
   [clojure-mcp.dialects :as dialects]
   [taoensso.timbre :as log])
  (:import [java.io Closeable]))

(defn- get-state [service]
  (get service ::state))

(defn- connect [service]
  (let [{:keys [host port]} service]
    (nrepl/connect :host (or host "localhost") :port port)))

(defn open-connection
  "Opens an nREPL transport and client pair for the given service.
  Callers are responsible for closing via `close-connection`."
  ([service]
   (open-connection service Long/MAX_VALUE))
  ([service timeout-ms]
   (let [transport (connect service)
         client (nrepl/client transport timeout-ms)]
     {:transport transport
      :client client})))

(defn close-connection [{:keys [transport]}]
  (when transport
    (.close ^Closeable transport)))

(defn new-eval-id []
  (nrepl.misc/uuid))

(defn- get-stored-session [service session-type]
  (let [port (:port service)
        state @(get-state service)]
    (get-in state [:ports port :sessions session-type])))

(defn- update-stored-session! [service session-type session-id]
  (let [port (:port service)]
    (swap! (get-state service) assoc-in [:ports port :sessions session-type] session-id)))

(defn- session-valid? [client session-id]
  (try
    (let [sessions (-> (nrepl/message client {:op "ls-sessions"})
                       nrepl/combine-responses
                       :sessions)]
      (contains? (set sessions) session-id))
    (catch Exception _ false)))

(defn- ensure-session! [client service session-type]
  (let [stored-id (get-stored-session service session-type)]
    (if (and stored-id (session-valid? client stored-id))
      stored-id
      (let [new-id (nrepl/new-session client)]
        (update-stored-session! service session-type new-id)
        new-id))))

(defn current-ns
  "Returns the current namespace for the given session type."
  [service session-type]
  (let [port (:port service)
        state @(get-state service)]
    (get-in state [:ports port :current-ns session-type])))

(defn- update-current-ns! [service session-type new-ns]
  (let [port (:port service)]
    (swap! (get-state service) assoc-in [:ports port :current-ns session-type] new-ns)))

(def truncation-length 10000)

(defn eval-code*
  "Low-level eval helper that executes using an existing connection map.
  Returns a map containing the responses plus the session/id used."
  [service {:keys [client]} code {:keys [session-type session-id eval-id] :as _opts}]
  (let [session-type (or session-type :default)
        session-id (or session-id (ensure-session! client service session-type))
        eval-id (or eval-id (new-eval-id))
        msg {:op "eval"
             :code code
             :session session-id
             :id eval-id
             :nrepl.middleware.print/print "nrepl.util.print/pprint"
             :nrepl.middleware.print/quota truncation-length}
        responses (doall (nrepl/message client msg))]
    ;; Update current-ns if present in any response
    (doseq [resp responses]
      (when-let [new-ns (:ns resp)]
        (update-current-ns! service session-type new-ns)))
    {:responses responses
     :session-id session-id
     :eval-id eval-id
     :session-type session-type}))

(defn- eval-code-internal
  "Internal eval - opens connection, no initialization.
   Used by init code to avoid circular calls."
  [service code & {:keys [session-type]}]
  (let [conn (open-connection service)]
    (try
      (let [{:keys [responses]} (eval-code* service conn code {:session-type session-type})]
        responses)
      (finally
        (close-connection conn)))))

(defn describe
  "Returns the nREPL server's description, synchronously."
  [service]
  (with-open [conn (connect service)]
    (let [client (nrepl/client conn 10000)]
      (nrepl/combine-responses (nrepl/message client {:op "describe"})))))

(defmulti ^:private fetch-project-directory-helper
  "Helper multimethod for fetching project directory based on env-type."
  (fn [nrepl-env-type _service] nrepl-env-type))

(defmethod fetch-project-directory-helper :default [nrepl-env-type service]
  ;; default to fetching from the nrepl
  ;; Uses eval-code-internal to avoid triggering full init (we're called during init)
  (when-let [exp (dialects/fetch-project-directory-exp nrepl-env-type)]
    (try
      (let [result-value (->> (eval-code-internal service exp :session-type :tools)
                              nrepl/combine-responses
                              :value)]
        result-value)
      (catch Exception e
        (log/warn e "Failed to fetch project directory")
        nil))))

(defmethod fetch-project-directory-helper :scittle [_ service]
  (when-let [desc (describe service)]
    (some-> desc :aux :cwd io/file (.getCanonicalPath))))
