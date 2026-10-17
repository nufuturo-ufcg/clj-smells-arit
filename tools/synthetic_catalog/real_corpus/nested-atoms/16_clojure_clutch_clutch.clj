(ns com.ashafa.clutch
  (:require [com.ashafa.clutch [utils :as utils]]
            [cheshire.core :as json]
            [clojure.java.io :as io]
            [cemerick.url :as url]
            clojure.string)
  (:use com.ashafa.clutch.http-client)
  (:import (java.io File FileInputStream BufferedInputStream InputStream ByteArrayOutputStream)
           (java.net URL))
  (:refer-clojure :exclude (conj! assoc! dissoc!)))

(defdbop database-info
  [db]
  (couchdb-request :get db)
  #_(when-let [info (couchdb-request :get db)]
    (merge info
           (when-let [watchers (@watched-databases (str db))]
             {:watchers (keys watchers)}))))

(defdbop changes
  "Returns a lazy seq of the rows in _changes, as configured by the given options.

   If you want to react to change notifications, you should probably use `change-agent`."
  [db & {:keys [since limit descending feed heartbeat
                timeout filter include_docs style] :as opts}]
  (let [url (url/url db "_changes")
        response (couchdb-request* :get (assoc url :query opts :as :stream))]
    (when-not response
      (throw (IllegalStateException. (str "Database for _changes feed does not exist: " url))))
    (-> response
      :body
      (lazy-view-seq (not= "continuous" feed))
      (vary-meta assoc ::http-resp response))))

(defn- change-agent-config
  [db options]
  (merge {:heartbeat 30000 :feed "continuous"}
         options
         (when-not (:since options)
           {:since (:update_seq (database-info db))})
         {::db db
          ::state :init
          ::last-update-seq nil}))

(defdbop change-agent
  "Returns an agent whose state will very to contain events
   emitted by the _changes feed of the specified database.

   Users are expected to attach functions to the agent using
   `add-watch` in order to be notified of changes.  The
   returned change agent is entirely 'managed', with
   `start-changes` and `stop-changes` controlling its operation
   and sent actions.  If you send actions to a change agent, 
   bad things will likely happen."
  [db & {:as opts}]
  (agent nil :meta {::changes-config (atom (change-agent-config db opts))}))

(defn- run-changes
  [_]
  (let [config-atom (-> *agent* meta ::changes-config)
        config @config-atom]
    (case (::state config)
      :init (let [changes (apply changes (::db config) (flatten (remove (comp namespace key) config)))
                  http-resp (-> changes meta ::http-resp)]
              ; cannot shut down continuous _changes feeds without aborting this
              (assert (-> http-resp :request :http-req))
              (swap! config-atom merge {::seq changes
                                        ::http-resp http-resp
                                        ::state :running})
              (send-off *agent* #'run-changes)
              nil)
      :running (let [change-seq (::seq config)
                     change (first change-seq)
                     last-change-seq (or (:seq change) (:last_seq change))]
                 (send-off *agent* #'run-changes)
                 (when-not (= :stopped (::state @config-atom))
                   (swap! config-atom merge
                          {::seq (rest change-seq)}
                          (when last-change-seq {::last-update-seq last-change-seq})
                          (when-not change {::state :stopped}))
                   change))
      :stopped (-> config ::http-resp :request :http-req .abort))))

(defn start-changes
  "Starts the flow of changes through the given change-agent.
   All of the options provided to `change-agent` are used to
   configure the underlying _changes feed."
  [change-agent]
  (send-off change-agent #'run-changes))

(defn stop-changes
  "Stops the flow of changes through the given change-agent.
   Change agents can be restarted with `start-changes`."
  [change-agent]
  (swap! (-> change-agent meta ::changes-config) assoc ::state :stopped)
  change-agent)
