(ns ^{:doc "Defines protocols for graphs, digraphs, and weighted graphs.
Also provides record implementations and constructors for simple graphs --
weighted, unweighted, directed, and undirected. The implementations are based
on adjacency lists."
      :author "Justin Kramer"}
  loom.graph
  (:require [loom.alg-generic :refer [bf-traverse]]
            #?@(:clj [[loom.cljs :refer (def-protocol-impls)]]))
  #?@(:cljs [(:require-macros [loom.cljs :refer [def-protocol-impls extend]])]))

(defprotocol Graph
  (nodes [g] "Returns a collection of the nodes in graph g")
  (edges [g] "Edges in g. May return each edge twice in an undirected graph")
  (has-node? [g node] "Returns true when node is in g")
  (has-edge? [g n1 n2] "Returns true when edge [n1 n2] is in g")
  (successors* [g node] "Returns direct successors of node")
  (out-degree [g node] "Returns the number of outgoing edges of node")
  (out-edges [g node] "Returns all the outgoing edges of node"))

(defprotocol WeightedGraph
  (weight* [g e] [g n1 n2] "Returns the weight of edge e or edge [n1 n2]"))

(defn successors
  "Returns direct successors of node"
  ([g] #(successors g %)) ; faster than partial
  ([g node] (successors* g node)))

(defn predecessors
  "Returns direct predecessors of node"
  ([g] #(predecessors g %))
  ([g node] (predecessors* g node)))

(defn weight
 "Returns the weight of edge e or edge [n1 n2]"
  ([g] (partial weight g))
  ([g e] (weight* g (src e) (dest e)))
  ([g n1 n2] (weight* g n1 n2)))

(defn add-nodes
  "Adds nodes to graph g. Nodes can be any type of object"
  [g & nodes]
  (add-nodes* g nodes))

(defn add-edges
  "Adds edges to graph g. For unweighted graphs, edges take the form [n1 n2].
  For weighted graphs, edges take the form [n1 n2 weight] or [n1 n2], the
  latter defaulting to a weight of 1"
  [g & edges]
  (add-edges* g edges))

(defrecord BasicEditableGraph [nodeset adj])

(def ^{:dynamic true
       :doc "Weight used when none is given for edges in weighted graphs"}
  *default-weight* 1)

(extend BasicEditableGraph
  Graph
  (merge default-all default-unweighted)

  EditableGraph
  {:add-nodes*
   (fn [g nodes]
     (reduce
      (fn [g node] (update-in g [:nodeset] conj node))
      g nodes))

   :add-edges*
   (fn [g edges]
     (reduce
      (fn [g [n1 n2]]
        (-> g
            (update-in [:nodeset] conj n1 n2)
            (update-in [:adj n1] (fnil conj #{}) n2)
            (update-in [:adj n2] (fnil conj #{}) n1)))
      g edges))

   :remove-nodes*
   (fn [g nodes]
     (let [nbrs (mapcat #(successors g %) nodes)]
       (-> g
           (update-in [:nodeset] #(apply disj % nodes))
           (assoc :adj (remove-adj-nodes (:adj g) nodes nbrs disj)))))

   :remove-edges*
   (fn [g edges]
     (reduce
      (fn [g [n1 n2]]
        (-> g
            (update-in [:adj n1] disj n2)
            (update-in [:adj n2] disj n1)))
      g edges))

   :remove-all
   (fn [g]
     (assoc g :nodeset #{} :adj {}))})

(defrecord FlyGraph [fnodes fedges fsuccessors start])

(defrecord FlyDigraph [fnodes fedges fsuccessors fpredecessors start])

(defrecord WeightedFlyGraph [fnodes fedges fsuccessors fweight start])

(defrecord WeightedFlyDigraph
    [fnodes fedges fsuccessors fpredecessors fweight start])

(extend FlyGraph
  Graph default-flygraph-graph-impl)

(extend FlyDigraph
  Graph default-flygraph-graph-impl
  Digraph default-flygraph-digraph-impl)

(extend WeightedFlyGraph
  Graph default-flygraph-graph-impl
  WeightedGraph default-flygraph-weighted-impl)

(extend WeightedFlyDigraph
  Graph default-flygraph-graph-impl
  Digraph default-flygraph-digraph-impl
  WeightedGraph default-flygraph-weighted-impl)

(defn graph?
  "Returns true if g satisfies the Graph protocol"
  [g]
  (satisfies? Graph g))

(defn weighted?
  "Returns true if g satisfies the WeightedGraph protocol"
  [g]
  (satisfies? WeightedGraph g))

(defn build-graph
  "Builds up a graph (i.e. adds edges and nodes) from any combination of
  other graphs, adjacency maps, edges, or nodes."
  [g & inits]
  (letfn [(build [g init]
            (cond
             ;; graph
             (graph? init)
             (if (and (weighted? g) (weighted? init))
               (assoc
                   (reduce add-edges
                           (add-nodes* g (nodes init))
                           (for [[n1 n2] (edges init)]
                             [n1 n2 (weight init n1 n2)]))
                 :attrs (merge (:attrs g) (:attrs init)))
               (-> g
                   (add-nodes* (nodes init))
                   (add-edges* (edges init))
                   (assoc :attrs (merge (:attrs g) (:attrs init)))))
             ;; adacency map
             (map? init)
             (let [es (if (map? (val (first init)))
                        (for [[n nbrs] init
                              [nbr wt] nbrs]
                          [n nbr wt])
                        (for [[n nbrs] init
                              nbr nbrs]
                          [n nbr]))]
               (-> g
                   (add-nodes* (keys init))
                   (add-edges* es)))
             ;; edge
             (sequential? init) (add-edges g init)
             ;; node
             :else (add-nodes g init)))]
    (reduce build g inits)))

(defn graph
  "Creates an unweighted, undirected graph. inits can be edges, adjacency maps,
  or graphs"
  [& inits]
  (apply build-graph (BasicEditableGraph. #{} {}) inits))

(defn fly-graph
  "Creates a read-only, ad-hoc graph which uses the provided functions
  to return values for nodes, edges, etc. If any members are not functions,
  they will be returned as-is. Edges can be inferred if nodes and
  successors are provided. Nodes and edges can be inferred if successors and
  start are provided."
  [& {:keys [nodes edges successors predecessors weight start]}]
  (cond
   (and predecessors weight)
   (WeightedFlyDigraph. nodes edges successors predecessors weight start)
   predecessors
   (FlyDigraph. nodes edges successors predecessors start)
   weight
   (WeightedFlyGraph. nodes edges successors weight start)
   :else
   (FlyGraph. nodes edges successors start)))
