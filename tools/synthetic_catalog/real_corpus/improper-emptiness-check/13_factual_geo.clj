(ns geo.h3
  "Working with H3."
  (:require [geo.crs :as crs]
            [geo.spatial :as spatial]
            [geo.jts :as jts]
            [clojure.string :as string]
            [clojure.walk :as walk]
            [clojure.math.numeric-tower :as numeric-tower])
  (:import (ch.hsr.geohash GeoHash)
           (com.uber.h3core AreaUnit H3Core LengthUnit)
           (com.uber.h3core.util GeoCoord)
           (geo.spatial Point Shapelike)
           (org.locationtech.jts.geom Geometry LinearRing MultiPolygon Polygon)
           (org.locationtech.spatial4j.shape.impl RectangleImpl)))

(def ^H3Core h3-inst (H3Core/newInstance))

(def ^:internal safe-polyfill-hexagon-maximum
  "Lowering this will reduce the likelihood of H3's polyfill crashing, but
  will cause this library's polyfill to run more slowly due to more frequent subdivision"
  60000)

(defn pt->h3
  "Return the Long index of the resolution 'res' cell that a point or lat/lng pair is contained within."
  ([^Point pt ^Integer res]
   (pt->h3 (spatial/latitude pt) (spatial/longitude pt) res))
  ([^Double lat ^Double lng ^Integer res]
   (try (.geoToH3 h3-inst lat lng res)
        (catch Exception e
          (throw (Exception. (string/join ["Failed to complete pt->h3 for lat " lat ", long " lng, ", res " res ". H3 exception message: " e])))))))

(defn- hex-radius-in-meters
  "See h3's bbox.c."
  [pt res]
  (let [h (pt->h3 pt res)
        h-jts (h3->pt h)
        h-centroid (spatial/point h-jts)
        h-boundary (to-jts h)
        c1 (first (.getCoordinates ^Geometry h-boundary))
        s1 (spatial/to-spatial4j-point c1)]
    (spatial/distance h-centroid s1)))

(defn- max-kring-size
  "Maximum indices that result from k-ring with a given k. See h3's algos.c."
  [k]
  (->> (for [i (range k)
             :let [r (* 6 (+ i 1))]]
         r)
       (reduce +)
       inc))

(defn- max-polyfill-size
  "Maximum number of indices used by h3's circular/k-ring polyfill method.
  See h3's algos.c. If h3's polyfill method is made more efficient, this should
  also change accordingly."
  [shape ^Integer res]
  (let [bbox (spatial/bounding-box shape)
        closest-side-vertex (if (< (Math/abs (.getMinY bbox))
                                   (Math/abs (.getMaxY bbox)))
                                (spatial/point (.getMinY bbox) (.getMinX bbox))
                                (spatial/point (.getMaxY bbox) (.getMaxY bbox)))

        centroid (spatial/center bbox)
        bbox-radius-m (spatial/distance closest-side-vertex centroid)
        center-hex-radius-m (hex-radius-in-meters (h3->pt (pt->h3 centroid res)) res)
        bbox-hex-radius (Math/ceil (/ bbox-radius-m (* 1.5 center-hex-radius-m)))]
    (max-kring-size bbox-hex-radius)))

(defn- polyfill-check-common
  "Common logic needed to see if a all shapes in a set are single polygons that
  are sufficiently small to be polyfilled. If h3's polyfill method is made more efficient,
  modifying max-polyfill-size accordingly should make this run faster."
  [shapes ^Integer res]
  (loop [good-geoms []
         remaining-geoms shapes]
    (if (not (empty? remaining-geoms))
      (let [current-poly (first remaining-geoms)
            max-hexagons (if (.isEmpty (spatial/to-jts current-poly))
                           0
                           (max-polyfill-size current-poly res))]
        (cond (zero? max-hexagons)
              ; If max-hexagons yields zero, the shape is either too small or invalid, so skip.
              (recur good-geoms (rest remaining-geoms))
              (instance? MultiPolygon current-poly)
              ; If a subdivided quadrant became a multipolygon, split that into its polygons and recur
              (recur good-geoms (concat (jts/geometries current-poly) (rest remaining-geoms)))
              (< max-hexagons safe-polyfill-hexagon-maximum)
              ; If less than the safe maximum, the geometry is good to be polyfilled
              (recur (conj good-geoms current-poly) (rest remaining-geoms))
              :else
              ; Otherwise, if it's a polygon but larger than the safe maximum, divide into quadrants and recur
              (recur good-geoms (concat (jts/subdivide (spatial/to-jts current-poly)) (rest remaining-geoms)))))
      good-geoms)))
