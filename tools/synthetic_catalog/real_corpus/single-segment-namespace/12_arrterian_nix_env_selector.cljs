(ns config
  (:require [vscode.workspace :as workspace]
            [utils.helpers :refer [render-workspace]]
            [utils.logger :as logger]))

(defonce config (atom {}))
