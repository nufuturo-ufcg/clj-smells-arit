(ns implicit-namespace-deps-cljs-example
  (:require [foo.widgets :refer :all]
            [foo.tools :as tools]
            [foo.explicit :refer [specific]]))

;; Host objects are globals in ClojureScript and are not namespace dependencies.
(defn host-interop []
  (js/console.log (.-innerWidth js/window))
  (.stringify js/JSON {:ok true}))

;; Explicit aliases and referred symbols are visible dependencies.
(defn explicit-dependencies [value]
  (tools/normalize (specific value)))

;; An explicit :only list is not an implicit import.
(use '[foo.explicit :only [specific]])

;; Qualified project symbols are intentionally left for the cross-namespace pass.
(defn unresolved-qualified-call []
  (project.runtime/start! {:mode :default}))
