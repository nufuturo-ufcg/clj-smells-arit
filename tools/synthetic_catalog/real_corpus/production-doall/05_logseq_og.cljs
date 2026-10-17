(ns frontend.handler.file
  "Provides util handler fns for files"
  (:refer-clojure :exclude [load-file])
  (:require [frontend.config :as config]
            [frontend.db :as db]
            [frontend.fs :as fs]
            [frontend.fs.nfs :as nfs]
            [frontend.fs.capacitor-fs :as capacitor-fs]
            [frontend.handler.common.file :as file-common-handler]
            [frontend.handler.common.config-edn :as config-edn-common-handler]
            [frontend.handler.repo-config :as repo-config-handler]
            [frontend.handler.global-config :as global-config-handler]
            [frontend.handler.ui :as ui-handler]
            [frontend.schema.handler.global-config :as global-config-schema]
            [frontend.schema.handler.repo-config :as repo-config-schema]
            [frontend.state :as state]
            [frontend.util :as util]
            [logseq.graph-parser.util :as gp-util]
            [electron.ipc :as ipc]
            [lambdaisland.glogi :as log]
            [promesa.core :as p]
            [frontend.mobile.util :as mobile-util]
            [logseq.common.path :as path]
            [logseq.graph-parser.config :as gp-config]))

(defn load-file
  [repo-url path]
  (->
   (p/let [content (fs/read-file (config/get-repo-dir repo-url) path)]
     content)
   (p/catch
    (fn [e]
      (println "Load file failed: " path)
      (js/console.error e)))))

(defn load-multiple-files
  [repo-url paths]
  (doall
   (mapv #(load-file repo-url %) paths)))
