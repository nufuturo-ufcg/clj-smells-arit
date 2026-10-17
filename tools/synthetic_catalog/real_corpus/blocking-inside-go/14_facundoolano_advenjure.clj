(ns advenjure.ui.input
  (:require [clojure.string :as string]
            [clojure.core.async :refer [go <!]]
            [advenjure.items :refer [all-item-names]]
            [advenjure.rooms :refer [visible-name-mappings]]
            [advenjure.utils :refer [direction-mappings current-room room-as-item]])
  (:import [jline.console ConsoleReader]
           [jline.console.completer StringsCompleter ArgumentCompleter NullCompleter AggregateCompleter]))

(def console (ConsoleReader.))

(defn- prompt [gs]
  (let [prompt-fn (get-in gs [:configuration :prompt])
        p         (prompt-fn gs)]
    (.readLine console p)))

(defn prompt-value
  "Ask the user to enter a value"
  [prompt]
  (go (.readLine console prompt)))
