(ns mage.bot.dev-env
  (:require
   [babashka.fs :as fs]
   [babashka.process :as p]
   [clojure.edn :as edn]
   [clojure.java.io :as io]
   [clojure.string :as str]
   [mage.bot.env :as bot-env]
   [mage.bot.preflight :as preflight]
   [mage.color :as c]
   [mage.shell :as shell]
   [mage.util :as u]
   [table.core :as t]))

(def ^:private root-override (atom nil))

(defn- root [] (or @root-override u/project-root-directory))

(def bot-dirs
  "Bot-specific directories to create under .bot/ in the worktree."
  {"cibot"    [".bot/cibot"]
   "fixbot"   [".bot/fixbot/playwright/sessions" ".bot/fixbot/playwright/sockets"]
   "qabot"    [".bot/qabot"]
   "reprobot" [".bot/reprobot"]
   "uxbot"    [".bot/uxbot/screenshots"]})

(defn- create-bot-dirs!
  "Create all bot directories and .autobot/ with initial status file."
  [_bot-name]
  ;; Create all bot-specific dirs
  (doseq [[_ dirs] bot-dirs]
    (doseq [dir dirs]
      (fs/create-dirs (str (root) "/" dir))))
  ;; Create .bot/autobot/ and initial status file
  (fs/create-dirs (str (root) "/.bot/autobot"))
  (let [status-path (str (root) "/.bot/autobot/llm-status.txt")]
    (when-not (fs/exists? status-path)
      (spit status-path "Starting up")))
  (println (c/green "  Created bot directories")))
