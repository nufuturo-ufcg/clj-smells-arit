(ns proton.lib.helpers
  (:require-macros [cljs.core.async.macros :refer [go]])
  (:require [clojure.string :as string :refer [split upper-case lower-case join]]
            [cljs.nodejs :as node]
            [clojure.set :as cljset]
            [cljs.core.async :as async :refer [close! chan >! <!]]
            [proton.config.proton :as config]))

(def keystroke->keybinding-map
   {"/" :slash
    ":" :colon
    ";" :semicolon
    "'" :quote})

(defn keybinding->keystroke [keybinding]
  (let [keystroke (filter (comp #{(keyword keybinding)} keystroke->keybinding-map) (keys keystroke->keybinding-map))]
    (if-not (empty? keystroke)
      (first keystroke)
      keybinding)))
