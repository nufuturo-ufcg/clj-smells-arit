(ns app.main.ui.workspace.viewport.snap-distances
  (:require
   [app.common.data :as d]
   [app.common.data.macros :as dm]
   [app.common.files.helpers :as cph]
   [app.common.geom.rect :as grc]
   [app.common.geom.shapes :as gsh]
   [app.common.math :as mth]
   [app.common.types.shape.layout :as ctl]
   [app.main.refs :as refs]
   [app.main.snap :as ams]
   [app.main.ui.formats :as fmt]
   [beicon.v2.core :as rx]
   [clojure.set :as set]
   [cuerdas.core :as str]
   [rumext.v2 :as mf]))

(defn- query-worker
  [page-id coord [selrect selected frame]]
  (let [lt-side (if (= coord :x) :left :top)
        gt-side (if (= coord :x) :right :bottom)

        vbox  (deref refs/vbox)
        frame-sr (when-not (cph/root? frame) (dm/get-prop frame :selrect))
        bounds (d/nilv (grc/clip-rect frame-sr vbox) vbox)
        areas (gsh/get-areas bounds selrect)

        query-side
        (fn [side]
          (let [rect (get areas side)]
            (if (and (> (:width rect) 0) (> (:height rect) 0))
              (ams/select-shapes-area page-id (:id frame) selected @refs/workspace-page-objects rect)
              (rx/of nil))))]

    (rx/combine-latest (query-side lt-side)
                       (query-side gt-side))))
