(ns flatland.jiraph.graph
  (:use [clojure.string :only [split join]]
        [flatland.jiraph.utils :only [meta-id meta-id?]]
        [flatland.useful.map :only [filter-keys-by-val assoc-in* update-in* keyed]]
        [flatland.useful.utils :only [memoize-deref adjoin into-set map-entry verify]]
        [flatland.useful.fn :only [fix]]
        [flatland.ego.core :only [type-key]]
        [flatland.ordered.set :only [ordered-set]]
        [flatland.ordered.map :only [ordered-map]]
        flatland.useful.debug)
  (:require [flatland.jiraph.layer :as layer :refer [dispatch-update]]
            [flatland.jiraph.wrapped-layer :as wrapped]
            [flatland.retro.core :as retro]))

(defn edges
  "Gets edges from a node. Returns all edges, including deleted ones."
  [node]
  (:edges node))

(defn- ^:dynamic query-fn [layer keyseq not-found f]
  (when (or (not *read*)
            (= f identity))
    (layer/query-fn layer keyseq not-found f)))

(defn query-in-node*
  "Fetch data from inside a node, replacing it with not-found if it is missing,
   and immediately call a function on it."
  [layer keyseq not-found f & args]
  (verify (sequential? keyseq) (format "Can't look up keyseq %s in layer %s"
                                       (pr-str keyseq) (pr-str layer)))
  (if-let [query-fn (query-fn layer keyseq not-found f)]
    (apply query-fn args)
    (if-let [query-fn (query-fn layer keyseq not-found identity)]
      (apply f (query-fn) args)
      (let [[id & keys] keyseq
            node (get-node layer id nil)]
        (apply f (get-in node keys) args)))))

(defn get-in-node
  "Fetch data from inside a node."
  [layer keyseq & [not-found]]
  (query-in-node* layer keyseq not-found identity))

(defn get-in-edge
  "Fetch data from inside an edge."
  [layer [id to-id & keys] & [not-found]]
  (get-in-node layer (list* id :edges to-id keys) not-found))
