(ns pallet.ssh.execute
  "Execution of pallet actions via ssh"
  (:require
   [clojure.java.io :as io]
   [clojure.string :as string]
   [clojure.tools.logging :as logging]
   [pallet.action-impl :refer [action-symbol]]
   [pallet.action-plan :refer [context-label]]
   [pallet.common.filesystem :as filesystem]
   [pallet.common.logging.logutils :as logutils]
   [pallet.core.session :refer [effective-user]]
   [pallet.core.user :refer [effective-username obfuscated-passwords]]
   [pallet.execute :as execute
    :refer [clean-logs log-script-output result-with-error-map]]
   [pallet.local.execute :as local]
   [pallet.node :as node]
   [pallet.script-builder :as script-builder]
   [pallet.script.lib :as lib]
   [pallet.script.lib
    :refer [chgrp chmod chown env exit mkdir path-group path-owner]]
   [pallet.stevedore :as stevedore :refer [fragment]]
   [pallet.transport :as transport]
   [pallet.transport.local]
   [pallet.transport.ssh]
   [pallet.utils :refer [log-multiline]]))

(def ssh-connection (transport/factory :ssh {}))

(defn authentication
  "Return the user to use for authentication.  This is not necessarily the
  admin user (e.g. when bootstrapping, it is the image user)."
  [session]
  (logging/debugf "authentication %s"
                  (into {} (obfuscated-passwords (:user session))))
  {:user (:user session)})

(defn endpoint
  [session]
  (let [target-node (-> session :server :node)
        proxy (node/proxy target-node)]
    (if proxy
      {:server (or (:host proxy "localhost"))
       :port (:port proxy)}
      {:server (node/node-address target-node)
       :port (node/ssh-port target-node)})))

(def ssh-connection-options {:max-tries 3})

(defn get-connection [session]
  (transport/open
   ssh-connection (endpoint session) (authentication session)
   ssh-connection-options))

(defn release-connection [session connection]
  (assert connection)
  (transport/release
   ssh-connection connection (endpoint session) (authentication session)
   ssh-connection-options))

(defn expire-connection [session]
  (transport/expire
   ssh-connection (endpoint session) (authentication session)
   ssh-connection-options))

(defmacro ^{:indent 2} with-connection
  "Execute the body with a connection to the current target node,"
  [session [connection & [no-cache]] & body]
  `(with-connection* ~session
     (fn []
       (try
         (let [~connection (get-connection ~session)
               res# (do ~@body)]
           (assert ~connection)
           (when-not ~no-cache
             (release-connection ~session ~connection))
           res#)
         (catch Exception e#
           (expire-connection ~session)
           (with-connection-exception-handler e#))))))
