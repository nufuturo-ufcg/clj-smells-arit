(ns project.consumer.explicit
  (:require [project.runtime :as runtime]))

(defn call-runtime [options]
  (runtime/start! options))
