(ns implicit-namespace-deps-interop-example)

;; These nested forms are JavaScript method calls, not standalone namespace use.
(defn attach-middleware [app middleware]
  (. app (use middleware)))

(defn attach-nested-middleware [app middleware]
  (.. app (use middleware)))
