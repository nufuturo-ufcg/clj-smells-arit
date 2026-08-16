(ns relying-on-load-time-side-effects (:require [clojure.java.shell :as shell :refer [sh]]))

(def remote-config (slurp "https://example.test/config"))
(def server-socket (java.net.Socket. "localhost" 8080))

(def deferred-config (delay (slurp "https://example.test/config")))
(def pure-config {:port 8080})

(defn read-config [path]
  (slurp path))

;; Known eager forms preserve proof of load-time execution.
(def eager-nested
  (when true
    (spit "/tmp/arit-load-test" "value")))

;; An unresolved call can be an external macro. Its arguments are unknown.
(def route-definition
  (external.routes/GET "/config" []
    (slurp "https://example.test/config")))

;; A locally-defined macro is also unknown without macro expansion.
(defmacro postpone [& body]
  `(fn [] ~@body))

(def macro-produced-value
  (postpone (slurp "https://example.test/config")))

;; Syntax-quoted code is data, not an executed initializer.
(def syntax-quoted-template
  `(slurp "https://example.test/config"))

;; Asynchronous/deferred bodies are not synchronous load-time execution.
(def deferred-future
  (future (slurp "https://example.test/config")))

;; A local function with the same name is not clojure.core/slurp.
(defn slurp [value] value)
(def local-value (slurp "not-io"))

;; Namespace aliases and explicit refers resolve to the same canonical var.
(def aliased-shell-result (shell/sh "true"))
(def referred-shell-result (sh "true"))
