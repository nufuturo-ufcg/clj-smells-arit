(ns misuse-of-dynamic-scope-explicit)

(def ^:dynamic request-context nil)

(future
  (println request-context))

(future
  (let [request-context :local]
    (println request-context)))
