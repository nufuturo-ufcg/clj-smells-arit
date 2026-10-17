(ns io.github.humbleui.debug
  (:require
    [io.github.humbleui.canvas :as canvas]
    [io.github.humbleui.util :as util]
    [io.github.humbleui.font :as font]
    [io.github.humbleui.protocols :as protocols]
    [io.github.humbleui.signal :as signal])
  (:import
    [io.github.humbleui.jwm Window]
    [io.github.humbleui.skija Paint]))

(def width
  100)

(def *frame-idx
  (volatile! 0))

(defmacro frame-get [array idx]
  `(aget ~array (mod (+ ~idx @*frame-idx) width)))
