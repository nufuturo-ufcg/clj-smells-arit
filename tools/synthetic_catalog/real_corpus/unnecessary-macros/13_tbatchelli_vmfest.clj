(ns vmfest.virtualbox.conditions
  (:require
   [clojure.pprint :refer [pprint]]
   [clojure.stacktrace :refer [root-cause]]
   [clojure.tools.logging :as log]
   [slingshot.slingshot :refer [try+ throw+]]
   [vmfest.virtualbox.version :refer [evaluate-when]])
  (:import [org.virtualbox_4_3 VBoxException IVirtualBoxErrorInfo]))

(defn handle-vbox-runtime* [condition type-action-map]
  (let [error-type (:original-error-type condition)
        action (error-type type-action-map)]
    (if action action
        (throw+ condition))))

(defmacro handle-vbox-runtime [type-action-map]
  `(handle-vbox-runtime* *condition* ~type-action-map))
