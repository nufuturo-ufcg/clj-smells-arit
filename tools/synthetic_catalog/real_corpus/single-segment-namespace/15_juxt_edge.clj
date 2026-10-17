(ns ^{:clojure.tools.namespace.repl/load false} dev-extras
  "This namespace provides all of the general-purpose dev utilities used from a
  `dev` namespace.  These vars are all available from `dev`."
  (:require
   [clojure.test :refer [run-tests]]
   [edge.system :as system]
   [edge.system.meta :as system.meta]
   [integrant.repl]
   [integrant.repl.state]
   io.aviso.ansi))
