(ns cider.piggieback
  "nREPL middleware enabling the transparent use of a ClojureScript REPL with nREPL tooling.

  This is the public face of Piggieback. It has no hard dependency on
  ClojureScript: when ClojureScript is on the classpath the real implementation
  in `cider.piggieback.cljs` is loaded lazily on first use; otherwise the
  middleware is a no-op and `cljs-repl` reports the missing dependency.

  The session-state dynamic vars live here (rather than in the implementation
  namespace) because they are part of Piggieback's contract: other middleware,
  notably cider-nrepl, read them out of the session by their fully-qualified
  name. Keeping them here also lets the implementation namespace depend on this
  one without a load cycle."
  {:author "Chas Emerick"}
  (:require
   [nrepl.middleware :refer [set-descriptor!]]
   [nrepl.middleware.print :as print]))

(def ^:private cljs-available?
  "True when ClojureScript is on the classpath. Checked against `cljs.repl`
  directly (not the implementation namespace) so that resolving availability
  doesn't trigger a load cycle."
  (try
    (require 'cljs.repl)
    true
    (catch Throwable _ false)))
