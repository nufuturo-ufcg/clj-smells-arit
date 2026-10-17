(ns html-string
  (:require [nextjournal.clerk :as clerk]
            [nextjournal.clerk.eval :as eval]
            [nextjournal.clerk.view :as view]
            [nextjournal.clerk.viewer :as viewer]
            [clojure.repl.deps :as deps]))

(defonce html
  (do
    (deps/add-lib 'io.github.escherize/huff)
    @(requiring-resolve 'huff2.core/html)))
