(ns babashka.impl.pprint
  {:no-doc true}
  (:require [clojure.pprint :as pprint]
            [sci.core :as sci]
            [sci.pprint]
            [babashka.impl.clojure.core.async]))

(defonce patched? (volatile! false))

(def new-table-ize
  (fn [t m]
    (apply hash-map
           (mapcat
            #(when-let [v (get t (key %))] [v (val %)])
            m))))

(def pprint-ns (sci/create-ns 'clojure.pprint nil))

(def print-pprint-dispatch
  (sci/new-dynamic-var '*print-pprint-dispatch* pprint/*print-pprint-dispatch* {:ns pprint-ns}))

(def new-write
  (fn [object & kw-args]
    (let [options (merge {:stream true} (apply hash-map kw-args))]
      (with-bindings (new-table-ize pprint/write-option-table options)
        (with-bindings
          (if (or (not (= pprint/*print-base* 10)) pprint/*print-radix*)
            {#'pr @#'pprint/pr-with-base} {})
          (let [optval (if (contains? options :stream)
                         (:stream options)
                         true)
                base-writer (condp = optval
                              nil (java.io.StringWriter.)
                              true *out*
                              optval)]
            (if pprint/*print-pretty*
              (pprint/with-pretty-writer base-writer
                (pprint/write-out object))
              (binding [*out* base-writer]
                (pr object)))
            (if (nil? optval)
              (.toString ^java.io.StringWriter base-writer))))))))

(when-not @patched?
  (alter-var-root #'pprint/write (constantly new-write)))

(def current-length #'pprint/*current-length*)

(defn write-out
  "Write an object to *out* subject to the current bindings of the printer control
  variables. Use the kw-args argument to override individual variables for this call (and
  any recursive calls).
  *out* must be a PrettyWriter if pretty printing is enabled. This is the responsibility
  of the caller.
  This method is primarily intended for use by pretty print dispatch functions that
  already know that the pretty printer will have set up their environment appropriately.
  Normal library clients should use the standard \"write\" interface. "
  {:added "1.2"}
  [object]
  (let [length-reached (and
                        @current-length
                        @sci/print-length
                        (>= @current-length @sci/print-length))]
    (if-not pprint/*print-pretty*
      (pr object)
      (if length-reached
        (print "...")
        (do
          (when @current-length
            (.set ^clojure.lang.Var current-length (inc @current-length)))
          (print-pprint-dispatch object))))
    length-reached))
