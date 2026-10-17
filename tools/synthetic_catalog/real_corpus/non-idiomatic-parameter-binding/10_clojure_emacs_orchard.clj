(ns orchard.inspect
  "Clojure data structure inspection functionality.
  This code has a long history and at various points of
  time it lived in different projects. Originally
  it was part of swank-clojure, afterwards it was moved to
  javert, then forked to another project from which it
  was contributed to cider-nrepl. Finally cider-nrepl
  was split into two libraries and the code ended up here.

  Pretty wild, right?"
  (:refer-clojure :exclude [replace])
  (:require
   [clojure.reflect :as reflect]
   [clojure.set :as set]
   [clojure.string :as str]
   [orchard.inspect.analytics :as analytics]
   [orchard.java.compatibility :as compat]
   [orchard.pp :as pp]
   [orchard.print :as print])
  (:import
   (java.lang.reflect Constructor Field Method Modifier)
   (java.util Arrays List Map)
   (orchard.print Diff DiffColl)))

(defn- print-string
  "Print or pretty print the string `value`, depending on the view mode
  of the inspector."
  [{:keys [indentation pretty-print]} value]
  (if pretty-print
    (pp/pprint-str value {:indentation (or indentation 0)})
    (print/print-str value)))

(defn render
  "Append one or more rendered elements to the inspector's :rendered output."
  ([inspector value]
   (update inspector :rendered conj! value))
  ([inspector value & values]
   (reduce render (render inspector value) values)))

(defn render-onto
  "Append all elements in `coll` to the inspector's :rendered output."
  [inspector coll]
  (reduce render inspector coll))

(defn render-ln
  "Append a newline to the inspector's :rendered output."
  [inspector]
  (render inspector '(:newline)))

(defn- padding [{:keys [indentation]}]
  (when (and (number? indentation) (pos? indentation))
    (if (= indentation 2) "  " ;; Fastpath
        (String. (char-array indentation \space)))))

(defn- render-indent
  ([inspector]
   (if-let [padding (padding inspector)]
     (render inspector padding)
     inspector))
  ([inspector & values]
   (render-onto (render-indent inspector) values)))

(defn render-value
  "Render the given `value` and add it to the index. `value-role` can be provided
  to mark in the index that the value comes from a collection, and `value-key`
  stands for the key by which value can be retrieved from its collection.
  `display-value` string can be provided explicitly."
  ([inspector value] (render-value inspector value nil))
  ([inspector value {:keys [value-role value-key display-value]}]
   (let [{:keys [index]} inspector
         display-value (or display-value (print-string inspector value))
         expr (seq [:value display-value (count index)])]
     (-> inspector
         (update :index conj {:value value
                              :role value-role
                              :key value-key})
         (update :rendered conj! expr)))))

(defn render-indented-value
  "Render `value` as an inspectable element on its own indented line."
  [inspector value & [value-opts]]
  (-> inspector
      (render-indent)
      (render-value value value-opts)
      (render-ln)))
