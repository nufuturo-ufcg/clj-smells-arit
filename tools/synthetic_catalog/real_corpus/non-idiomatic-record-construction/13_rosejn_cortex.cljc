(ns cortex.tree
  (:require [clojure.core.matrix :as mat]
            [clojure.zip :as zip])
  (:import [java.util ArrayList Collection]))

(defrecord TreeNode [feature-index split target children])

(defn indexed-array
  [X indices]
  (let [vX (vec X)]
    (map #(get vX %) indices)))

(defn mode
  "Returns the most comment value in a seq of items."
  [items]
  (first (last (sort-by val (frequencies items)))))

(defn split-samples
  "Returns a pair of lazy seqs of row indices of X that fall either to the left or right of the
  given split value and feature dimension.

  [left-samples right-samples]
  "
  [X x-indices feature-index split-val]
  (reduce (fn [[le gt] x-index]
            (if (<= (mat/mget X x-index feature-index) split-val)
              [(conj le x-index) gt]
              [le (conj gt x-index)]))
          [[] []]
          x-indices))

(defn decision-tree*
  [X Y {:keys [n-samples n-features split-fn feature-indices sample-indices] :as options}]
  (let [labels (doall (indexed-array Y sample-indices))]
    (cond
      ; All remaining samples have the same class => leaf node
      (= 1 (count (set labels))) (TreeNode. nil nil (first labels) [])

      ; All features have been split on already => leaf node
      (empty? feature-indices)   (TreeNode. nil nil (mode labels) [])

      ; TODO:
      ; - Add support for other stopping modes:
      ;   * max-depth: max depth of the tree
      ;   * max-leaves: max number of leaves in the tree
      ;   * min-branch-samples: min number of samples to still branch
      ;   * max-features: sample up to max-features without replacement to find
      ;   the best dimension on which to split at each node
      ;   * min-leaf-weight: if using weighted samples make a leaf when weight
      ;   combined sample weight at a node drops below a certain value

      ; Determine the next feature and split-val, then point to the children nodes
      :else
      (let [[feature-index split-val] (split-fn X Y feature-indices sample-indices options)
            feature-indices (filter #(not= feature-index %) feature-indices)
            [left-indices right-indices] (split-samples X sample-indices feature-index split-val)
            left-tree  (decision-tree* X Y (assoc options
                                                  :feature-indices feature-indices
                                                  :sample-indices left-indices))
            right-tree (decision-tree* X Y (assoc options
                                                  :feature-indices feature-indices
                                                  :sample-indices right-indices))
            root (TreeNode. feature-index split-val nil [left-tree right-tree])]
        root))))
