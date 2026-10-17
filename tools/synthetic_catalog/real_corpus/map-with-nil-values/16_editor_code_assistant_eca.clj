(ns eca.main
  (:refer-clojure :exclude [run!])
  (:gen-class)
  (:require
   [babashka.cli :as cli]
   [borkdude.dynaload]
   [clojure.string :as string]
   [eca.client-http :as client]
   [eca.config :as config]
   [eca.logger :as logger]
   [eca.network :as network]
   [eca.read-chat :as read-chat]
   [eca.server :as server]))

(defn ^:private version []
  (str "eca " (config/eca-version)))

(defn ^:private help
  [options-summary]
  (->> ["ECA - Editor Code Assistant"
        ""
        "Usage: eca <command> [<options>]"
        ""
        "All options:"
        options-summary
        ""
        "Available commands:"
        "  server        Start eca as server, listening to stdin."
        "  read-chat     Stream chat history from the DB cache as JSONL."
        ""
        "See https://eca.dev/config/introduction/ for detailed documentation."]
       (string/join \newline)))

(def log-levels #{"error" "warn" "info" "debug"})

(def cli-spec
  {:order [:help :version :verbose :config-file :log-level]
   :spec {:help {:alias :h
                 :desc "Print the available commands and its options"}
          :version {:desc "Print eca version"}
          :log-level {:ref "<LEVEL>"
                      :desc "The log level of eca logs, accepts. Defaults to 'info'."
                      :default "info"
                      :validate {:pred log-levels
                                 :ex-msg (fn [{:keys [_option _value]}]
                                           (format "Must be in %s" log-levels))}}
          :config-file {:ref "<FILE>"
                        :desc "Path to a JSON config <FILE> to use instead of searching default locations"
                        :coerce :string
                        :default nil}
          :verbose {:desc "Enable verbose JSON-RPC protocol tracing"}}})
