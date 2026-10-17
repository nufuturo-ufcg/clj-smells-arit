(ns classimg
  (:require [clojure2d.pixels :as p]
            [clojure2d.core :refer :all]
            [clojure2d.color :as c]
            [fastmath.classification :as cl]
            [fastmath.clustering :as clust]
            [fastmath.distance :as dist]
            [fastmath.vector :as v]
            [fastmath.core :as m]
            [fastmath.rbf :as rbf]
            [fastmath.random :as r]
            [clojure2d.extra.utils :as utl]))

(def img (p/load-pixels "generateme/blue.png"))

(def imgt (p/load-pixels "/mnt/hgfs/clojure-dropbox/mlimg/s.jpg"))

(def imgt img)

(def clusters (clust/k-means img 40))

(def labels (vec (:representatives clusters)))

(defn surrounding-idxs
  [^long x ^long y] 
  [[(dec x) (dec y)]
   [x (dec y)]
   [(inc x) (dec y)]
   [(dec x) y]
   ;; [(inc x) y]
   [(dec x) (inc y)]
   ;; [x (inc y)]
   ;; [(inc x) (inc y)]
   ;; [x y]
   ])

(defn color-vector
  [img ^long x ^long y]
  (map #(/ ^double % 255.0) (mapcat (fn [[x y]] (take 3 (p/get-color img x y))) (surrounding-idxs x y))))

(def samples (take 600 (map (fn [[^double x ^double y]]
                              (let [x (int (* x (width img)))
                                    y (int (* y (height img)))
                                    p (p/get-color img x y)
                                    l (labels (clusters p))]
                                [(color-vector img x y) l])) (r/sequence-generator :sobol 2))))

(def data (map first samples))

(def data-labels (map second samples))

(def cl (cl/lda {:max-nodes 10 :number-of-trees 10} data data-labels))

(def c (canvas (width imgt) (height imgt)))

(def p (p/clone-pixels imgt))

(def mapx (m/make-norm 0 (width img) 0 1))

(def mapy (m/make-norm 0 (height img) 0 1))

(def sampl (for [x (range 0 (width img) 10)
                 y (range 0 (height img) 10)]
             [(mapx x) (mapy y) (p/get-color img x y)]))

(def labels (map #(drop 2 %) sampl))

(def cl (cl/lda (map #(take 2 %) sampl) labels))

(def c (canvas (width imgt) (height imgt)))
