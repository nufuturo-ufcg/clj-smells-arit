(ns clojure2d.core-example
  (:require [clojure2d.core :refer :all]
            [metadoc.examples :refer :all]
            [fastmath.random :as r]
            [fastmath.core :as m]
            [fastmath.grid :as grid]
            [fastmath.vector :as v]
            [clojure2d.color :as c]))

(defsnippet clojure2d.core process-image-snippet
  "Process image with function `f` and save."
  (let [img (load-image "docs/cockatoo.jpg")
        unique-name (str "images/core/" (first opts) ".jpg")]
    (binding [*jpeg-image-quality* 0.7]
      (save (f img) (str "docs/" unique-name)))
    (str "../" unique-name)))

(defsnippet clojure2d.core draw-on-oriented-canvas
  "Draw axes on reoriented canvas"
  (let [canvas (canvas 400 200) 
        unique-name (str "images/core/" (first opts) ".jpg")]
    (with-oriented-canvas-> f canvas
      (set-background 0x30426a)
      (set-color :white)
      (line 5 5 150 5)
      (line 5 5 5 100)
      (line 150 5 145 10)
      (line 5 100 10 95)
      (line 150 5 145 0)
      (line 5 100 0 95)
      (text "X" 150 20)
      (text "Y" 15 100)
      (text "warning: text is reoriented" 20 40))
    (binding [*jpeg-image-quality* 0.85]
      (save canvas (str "docs/" unique-name)))
    (str "../" unique-name)))

(defsnippet clojure2d.core drawing-snippet
  "Draw something on canvas and save."
  (let [canvas (canvas 200 200) 
        unique-name (str "images/core/" (first opts) ".jpg")]
    (with-canvas-> canvas
      (set-background 0x30426a)
      (set-color :white)
      (f))
    (binding [*jpeg-image-quality* 0.85]
      (save canvas (str "docs/" unique-name)))
    (str "../" unique-name)))
