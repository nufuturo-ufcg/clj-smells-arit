(ns com.fulcrologic.fulcro.routing.legacy-ui-routers
  "Routers from Fulcro 2. These are a bit harder to use than the new dynamic router, and should probably not be used
   in new applications; however, they will be supported for the forseeable future."
  #?(:cljs (:require-macros com.fulcrologic.fulcro.routing.legacy-ui-routers))
  (:require
    [clojure.spec.alpha :as s]
    [com.fulcrologic.fulcro.mutations :as m :refer [defmutation]]
    [com.fulcrologic.fulcro.application :as app]
    [com.fulcrologic.fulcro.components :as comp :refer [defsc]]
    [com.fulcrologic.fulcro.algorithms.merge :as merge]
    [com.fulcrologic.fulcro.algorithms.normalize :as fnorm]
    [com.fulcrologic.fulcro.algorithms.do-not-use :as util]
    ;; babashka can't load the cljs compiler; :bb omits cljs.analyzer (error site below falls back to ex-info).
    #?@(:bb   []
        :clj  [[cljs.analyzer :as ana]]
        :cljs [["react" :as react]
               [goog.async.Deferred]
               [cljs.loader :as loader]])
    [taoensso.timbre :as log]))

(defmulti get-dynamic-router-target
  "Get the component that renders the given screen type. The parameter is simply the keyword of the module/component.
  Note that all have to match: the module name in the compiler that contains the code for the component,
  the first element of the ident returned by the component, and the keyword passed to this multimethod to retrieve
  the component."
  (fn [k] k))

(defmethod get-dynamic-router-target :default [k] nil)

(defn add-route-state [state-map target-kw component]
  (let [tree-state       {:tmp/new-route (comp/get-initial-state component)}
        query            [{:tmp/new-route (comp/get-query component)}]
        normalized-state (-> (fnorm/tree->db query tree-state true (merge/pre-merge-transform state-map))
                           (dissoc :tmp/new-route))]
    (util/deep-merge state-map normalized-state)))

(defn- install-route-impl [state target-kw component]
  (defmethod get-dynamic-router-target target-kw [k] component)
  (swap! state add-route-state target-kw component))
