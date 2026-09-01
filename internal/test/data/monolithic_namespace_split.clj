(ns monolithic-namespace-split.example
  (:import
   (example SchemaLoader)))

;; ========== SHOULD BE DETECTED ==========

;; Classic split: load pulls another file into the same compilation story
(load "core_extra")

(defn -main [& args]
  (println args))

;; Imperative namespace switch (often in a second file without ns)
(in-ns 'monolithic-namespace-split.example)

(defn temp [])

;; Qualified core symbols
(clojure.core/load "other_piece")
(clojure.core/in-ns 'some.other.ns)

(defn nested []
  ;; load inside a function body is still imperative stitching
  (load "runtime_patch"))

;; ========== SHOULD NOT BE DETECTED (inside comment) ==========

(comment
  (load "not-executed")
  (in-ns 'not.really))

;; Java static methods and similarly named namespace vars are not core/load.
(SchemaLoader/load "schema.json")
(example.loader/load "application-data")

;; A local definition shadows clojure.core/load.
(defn load [value] value)
(load "local-value")

;; A lexical binding shadows clojure.core/in-ns.
(let [in-ns identity]
  (in-ns 'local-value))
