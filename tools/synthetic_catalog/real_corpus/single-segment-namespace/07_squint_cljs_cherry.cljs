(ns browser-repl-test
  "End-to-end test of the cherry browser REPL, written in cherry.

  Spawns `vite dev` (which starts the nREPL server + serves the app), loads the
  page with a headless browser via playwright, connects an nREPL client over
  bencode TCP, and evaluates forms. Exercises the whole stack: compiler REPL
  output, /@resolve-deps ns fallback, nREPL bencode, vite-WS browser eval, and
  cross-ns redef visibility (the live-global alias binding)."
  (:require ["node:child_process" :as cp]
            ["node:fs" :as fs]
            ["node:path" :as path]
            ["playwright$default" :as pw]
            [clojure.string :as str]
            [nrepl-client :refer [make-client nrepl-request nrepl-eval msg-field with-timeout]]))
