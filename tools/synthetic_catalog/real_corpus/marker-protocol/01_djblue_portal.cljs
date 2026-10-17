(ns ^:no-doc portal.ui.rpc.runtime
  (:require [clojure.pprint :as pprint]
            [portal.runtime.cson.core :as core]
            [portal.runtime.cson.writer :as writer]
            [portal.ui.state :as state]
            [reagent.core :as r]))

(defprotocol Runtime)
