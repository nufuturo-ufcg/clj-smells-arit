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

(defn mcp-metadata-cache-file [] (fs/file (get-prompts-dir) "mcp-metadata-cache.edn"))

(def mcp-metadata-cache (atom {}))

(def cache-channel (async/chan))

(async/go-loop []
  (let [[k v] (async/<! cache-channel)
        updated-cache (swap! mcp-metadata-cache assoc k v)]
    (spit
     (mcp-metadata-cache-file)
     (pr-str updated-cache))
    (recur)))
