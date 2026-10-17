(ns clojure-lsp.server
  (:require
   [clojure-lsp.clojure-coercer :as clojure-coercer]
   [clojure-lsp.db :as db]
   [clojure-lsp.feature.command :as f.command]
   [clojure-lsp.feature.file-management :as f.file-management]
   [clojure-lsp.feature.semantic-tokens :as semantic-tokens]
   [clojure-lsp.feature.test-tree :as f.test-tree]
   [clojure-lsp.handlers :as handler]
   [clojure-lsp.logger :as logger]
   [clojure-lsp.nrepl :as nrepl]
   [clojure-lsp.producer :as producer]
   [clojure-lsp.settings :as settings]
   [clojure-lsp.shared :as shared]
   [clojure-lsp.startup :as startup]
   [clojure.core.async :as async]
   [jsonrpc4clj.coercer :as jsonrpc.coercer]
   [jsonrpc4clj.io-server :as jsonrpc.io-server]
   [jsonrpc4clj.liveness-probe :as jsonrpc.liveness-probe]
   [jsonrpc4clj.server :as jsonrpc.server]
   [lsp4clj.coercer :as coercer]
   [lsp4clj.lsp.requests :as lsp.requests]
   [medley.core :as medley]
   [promesa.core :as p]
   [promesa.exec :as p.exec]
   [taoensso.timbre :as timbre]
   [taoensso.timbre.appenders.community.otlp :as timbre.otlp])
  (:import
   [io.opentelemetry.sdk.autoconfigure AutoConfiguredOpenTelemetrySdk]
   [java.util.function Function]))

(defn ^:private setup-dev-environment [db* components]
  ;; We don't have an ENV=development flag, so the next best indication that
  ;; we're in a development environment is whether we're able to start an nREPL.
  (when-let [nrepl-port (nrepl/setup-nrepl)]
    ;; Save the port in the db, so it can be reported in server-info.
    (swap! db* assoc :port nrepl-port)
    ;; Add components to db* so it's possible to manualy call funcstions
    ;; which expect specific components
    (swap! db* assoc-in [:dev :components] components)
    ;; In the development environment, make the db* atom available globally as
    ;; db/db*, so it can be inspected in the nREPL.
    (alter-var-root #'db/db* (constantly db*))))
