(ns clojure.core.rrb-vector.rrbt
  (:refer-clojure :exclude [array-for push-tail pop-tail new-path do-assoc])
  (:require [clojure.core.rrb-vector.protocols
             :refer [PSliceableVector -slicev
                     PSpliceableVector -splicev]]
            [clojure.core.rrb-vector.nodes
             :refer [regular? empty-node node-ranges overflow? last-range
                     regular-ranges first-child last-child remove-leftmost-child
                     replace-leftmost-child replace-rightmost-child
                     fold-tail new-path* index-of-nil]]
            [clojure.core.rrb-vector.trees
             :refer [push-tail pop-tail new-path do-assoc]]
            [clojure.core.rrb-vector.transients
             :refer [ensure-editable editable-root editable-tail push-tail!
                     pop-tail! do-assoc!]])
  (:require-macros [clojure.core.rrb-vector.macros :refer [dbg]]))
