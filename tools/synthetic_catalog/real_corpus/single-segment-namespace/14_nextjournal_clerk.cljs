(ns screenshots
  "Playwright script to Generate open graph preview images from a notebook's results.

  Run this via:

      cd ui_tests; yarn nbb -m screenshots --url http://localhost:7777 --out-dir screenshots

  For a REPL development workflow start a nbb nrepl server via:

      cd ui_tests; yarn nbb nrepl-server :port 1337"

  {:clj-kondo/config '{:lint-as {promesa.core/let clojure.core/let}}}
  (:require ["playwright$default" :as pw :refer [chromium]]
            [clojure.edn :as edn]
            [promesa.core :as p]
            [nbb.core :refer [await]]
            [babashka.cli :as cli]))
