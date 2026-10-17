(ns advenjure.ui.input
  (:require [clojure.string :as string]
            [clojure.core.async :refer [go <!]]
            [advenjure.items :refer [all-item-names]]
            [advenjure.rooms :refer [visible-name-mappings]]
            [advenjure.utils :refer [direction-mappings current-room room-as-item]])
  (:import [jline.console ConsoleReader]
           [jline.console.completer StringsCompleter ArgumentCompleter NullCompleter AggregateCompleter]))

(defn read-file [file]
  (go (read-string (slurp file))))
