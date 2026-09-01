(ns project.consumer.missing)

;; The provider exists in the project index, but this namespace omits :require.
(defn call-runtime [options]
  (project.runtime/start! options))
