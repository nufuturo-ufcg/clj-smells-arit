(ns lt.objs.bottombar
  "Provide bottombar object and associated behaviors"
  (:require [lt.object :as object]
            [lt.objs.tabs :as tabs]
            [lt.objs.animations :as anim]
            [lt.objs.canvas :as canvas]
            [lt.util.cljs :refer [->dottedkw]]
            [lt.util.style :refer [->px]]
            [singultus.binding :refer [map-bound bound subatom]])
  (:require-macros [lt.macros :refer [behavior defui]]))

(def min-height 30)

(behavior ::height!
          :triggers #{:height!}
          :throttle 16
          :reaction (fn [this e]
                      (when-not (= 0 (.-clientY e))
                        (let [win-height (.-innerHeight js/window)
                              height (max (- win-height (.-clientY e)) min-height)]
                          (object/raise tabs/multi :bottom! (- height (:height @this)))
                          (object/merge! this {:height height
                                               :max-height height})))
                      ))
