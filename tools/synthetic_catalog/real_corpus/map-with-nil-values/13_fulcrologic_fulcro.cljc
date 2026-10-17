(ns com.fulcrologic.fulcro.raw.application
  (:require
    [clojure.string :as str]
    [com.fulcrologic.fulcro.algorithms.denormalize :as fdn]
    [com.fulcrologic.fulcro.algorithms.do-not-use :as util]
    [com.fulcrologic.fulcro.algorithms.indexing :as indexing]
    [com.fulcrologic.fulcro.algorithms.lookup :as ah]
    [com.fulcrologic.fulcro.algorithms.merge :as merge]
    [com.fulcrologic.fulcro.algorithms.normalize :as fnorm]
    [com.fulcrologic.fulcro.algorithms.scheduling :as sched]
    [com.fulcrologic.fulcro.algorithms.tempid :as tempid]
    [com.fulcrologic.fulcro.algorithms.tx-processing :as txn]
    [com.fulcrologic.fulcro.algorithms.tx-processing.synchronous-tx-processing :as stx]
    [com.fulcrologic.fulcro.data-fetch :as df]
    [com.fulcrologic.fulcro.mutations :as mut]
    [com.fulcrologic.fulcro.raw.components :as comp]
    [edn-query-language.core :as eql]
    #?@(:cljs [[goog.object :as gobj]
               [goog.functions :refer [debounce]]
               [goog.dom :as gdom]])
    [taoensso.timbre :as log])
  #?(:clj (:import (clojure.lang IDeref))))

(defn basis-t
  "Return the current basis time of the app."
  [app]
  (-> app :com.fulcrologic.fulcro.application/runtime-atom deref :com.fulcrologic.fulcro.application/basis-t))

(defn current-state
  "Get the current value of the application state database. If called without arguments it will attempt to find the app
   in the dynamically-bound comp/*app*, which is bound during render."
  [app-or-component]
  (let [app (comp/any->app app-or-component)]
    (-> app :com.fulcrologic.fulcro.application/state-atom deref)))

(defn tick!
  "Move the basis-t forward one tick. For internal use in internal algorithms. Fulcro
  uses this to add metadata to props so it can detect the newer of two version of props."
  [app]
  (swap! (:com.fulcrologic.fulcro.application/runtime-atom app) update :com.fulcrologic.fulcro.application/basis-t inc))

(defn update-shared!
  "Force shared props to be recalculated. This updates the shared props on the app, and future renders will see the
   updated values. This is a no-op if no shared-fn is defined on the app. If you're using React 16+ consider using
   Context instead of shared."
  [{:com.fulcrologic.fulcro.application/keys [runtime-atom] :as app}]
  (try
    (if-let [shared-fn (ah/app-algorithm app :shared-fn)]
      (let [shared       (-> app :com.fulcrologic.fulcro.application/runtime-atom deref :com.fulcrologic.fulcro.application/static-shared-props)
            state        (current-state app)
            root-class   (-> app :com.fulcrologic.fulcro.application/runtime-atom deref :com.fulcrologic.fulcro.application/root-class)
            query        (comp/get-query root-class state)
            v            (fdn/db->tree query state state)
            shared-props (merge shared (shared-fn v))]
        (swap! runtime-atom assoc :com.fulcrologic.fulcro.application/shared-props shared-props))
      (swap! runtime-atom assoc :com.fulcrologic.fulcro.application/shared-props
        (-> app :com.fulcrologic.fulcro.application/runtime-atom deref :com.fulcrologic.fulcro.application/static-shared-props)))
    (catch #?(:cljs :default :clj Throwable) e
      (log/error e "Cannot compute shared"))))

(defn root-props-changed?
  "Returns true if the props queries directly by the root component of the app (if mounted) have changed since the last
  render.  This is a shallow analysis such that, for example, a join from root (in a normalized db) will be checked as a difference
  of idents that the root prop points to.  This can be used for determining if things like shared-fn need to be re-run,
  and if it would simply be quicker to keyframe render the entire tree.

  This is a naivé algorithm that is essentially `select-keys` on the root props. It does not interpret the query in
  any way."
  [app]
  (let [{:com.fulcrologic.fulcro.application/keys [runtime-atom state-atom]} app
        {:com.fulcrologic.fulcro.application/keys [root-class indexes]} @runtime-atom]
    (if-not (comp/get-query root-class @state-atom)
      true
      (let [state-map       @state-atom
            prior-state-map (-> runtime-atom deref :com.fulcrologic.fulcro.application/last-rendered-state)
            root-props      (:root-props indexes)
            root-old        (select-keys prior-state-map root-props)
            root-new        (select-keys state-map root-props)]
        (not= root-old root-new)))))

(defn render!
  "Render the application immediately.  Prefer `schedule-render!`, which will ensure no more than 60fps.

  This is the central processing for render and cannot be overridden. `schedule-render!` will always invoke
  this function.  The optimized render is called by this function, which does extra bookkeeping and
  other supporting features common to all rendering.

  Options include:
  - `force-root?`: boolean.  When true disables all optimizations and forces a full root re-render.
  - anything your selected rendering optization system allows.  Shared props are updated via `shared-fn`
  only on `force-root?` and when (shallow) root props change.
  "
  ([app]
   (render! app {:force-root? false}))
  ([app {:keys [force-root?] :as options}]
   (tick! app)
   (let [{:com.fulcrologic.fulcro.application/keys [runtime-atom state-atom]} app
         {:com.fulcrologic.fulcro.application/keys [root-class]} (some-> runtime-atom deref)
         before-render (ah/app-algorithm app :before-render)
         sync-render!  (some-> (ah/app-algorithm app :vtx-render) deref)]
     (when before-render
       (try
         (before-render app root-class)
         (catch #?(:clj Throwable :cljs :default) e
           (log/error e "Transaction middleware threw an exception!"))))
     (when root-class
       (let [core-render!        (ah/app-algorithm app :core-render!)
             root-props-changed? (root-props-changed? app)]
         (when (or force-root? root-props-changed?)
           (update-shared! app))
         (cond
           (and (not force-root?) sync-render!) (sync-render!)
           core-render! (core-render! app (merge options {:root-props-changed? root-props-changed?})))
         (swap! runtime-atom assoc
           :com.fulcrologic.fulcro.application/last-rendered-state @state-atom
           :com.fulcrologic.fulcro.application/only-refresh #{}
           :com.fulcrologic.fulcro.application/to-refresh #{})))
     (let [batch-notifications (ah/app-algorithm app :batch-notifications)
           notify-all!         #(doseq [render-listener (-> runtime-atom deref :com.fulcrologic.fulcro.application/render-listeners vals)]
                                  (try
                                    (render-listener app options)
                                    (catch #?(:clj Exception :cljs :default) e
                                      (log/error e "Render listener failed. See https://book.fulcrologic.com/#err-render-listener-failed"))))]
       (if batch-notifications
         (batch-notifications notify-all!)
         (notify-all!))))))

(defn default-remote-error?
  "Default detection of network errors. Returns true if the status-code of the given result
  map is not 200."
  [{:keys [status-code]}]
  (not= 200 status-code))

(defn default-global-eql-transform
  "The default query transform function.  It makes sure the following items on a component query
  are never sent to the server:

  - Props whose namespace is `ui`
  - Any prop or join that is namespaced to com.fulcrologic.fulcro*
  - Any ident (as a prop or join) whose table name is namespaced ui or com.fulcrologic.fulcro*

  Takes an AST and returns the modified AST.
  "
  [ast]
  (let [kw-namespace (fn [k] (and (keyword? k) (namespace k)))]
    (df/elide-ast-nodes ast (fn [k]
                              (let [ns       (some-> k kw-namespace)
                                    ident-ns (when (eql/ident? k) (some-> (first k) kw-namespace))]
                                (or
                                  (and
                                    (string? ns)
                                    (or
                                      (= "ui" ns)
                                      (str/starts-with? ns "com.fulcrologic.fulcro.")))
                                  (and
                                    (string? ident-ns)
                                    (or
                                      (= "ui" ident-ns)
                                      (str/starts-with? ident-ns "com.fulcrologic.fulcro.")))))))))

(defn initialize-state!
  "Initialize the app state using `root` component's app state. This will deep merge against any data that is already
  in the state atom of the app. Can be called before `mount!`, in which case you should tell mount not to (re) initialize
  state."
  [app root]
  (when #?(:clj true :cljs goog.DEBUG)
    (comp/check-component-registry!))
  (let [initial-db   (-> app :com.fulcrologic.fulcro.application/state-atom deref)
        root-query   (comp/get-query root initial-db)
        _            (util/dev-check-query root-query comp/component-name)
        initial-tree (comp/get-initial-state root)
        db-from-ui   (if root-query
                       (-> (fnorm/tree->db root-query initial-tree true (merge/pre-merge-transform initial-tree))
                         (merge/merge-alternate-union-elements root))
                       initial-tree)
        db           (util/deep-merge initial-db db-from-ui)]
    (reset! (:com.fulcrologic.fulcro.application/state-atom app) db)))

(def ^:deprecated default-tx! txn/default-tx!)

(defrecord FulcroApp []
  Object
  (toString [_] "#<FulcroApp>"))

(defn fulcro-app
  "Create a new Fulcro application. See com.fulcrologic.fulcro.application/fulcro-app for the React-based initializer,
  which describes all supported options.

  This version creates an app that is not attached to React, and has no default root or optimized render. The
  map of initial options is the same *except* that react-centric options are obviously ignore, and also:

   * `:optimized-render!` - A `(fn ([app]) ([app txn-options]))` that can analyze the state of the application and optimally refresh the screen. Defaults to a no-op.
     This function is normally called from core-render!, and therefore is useless unless you define a `core-render!` that calls it.
   * `:core-render!` - A (fn [app txn-options] side-effect) that is called by schedule render. If you fail to supply this on a raw app, then
     NO rendering will happen; however, render listeners will still be called.
   * `:batch-notifications` - A side-effecting function `(fn [notify-all])` that can surround a batch of render notifiations with a context. The only
     argument to this function is `notify-all`, which is a function that will do the actual notifications. This is useful when using render
     notifications with React hooks, and need to tell React that a bunch of state changes need to happen together. The
     normal setting for this (if you're using hooks and render listeners) is
     `(fn [render!] (react-dom/unstable_batchedUpdates render!))` or
     `(fn [render!] (react-native/unstable_batchedUpdates render!))`.
   * `:refresh-component!` - A (fn [component]) that is used by synchronous transactions to attempt to refresh the UI
     of `component`. This has no definition for non-react applications, and cannot work without you defining this.

  Note that raw apps are not mounted, but are instead ready to be used immediately.  If you want to use inspect, then
  you must call `(inspect/client-started! app)` yourself.

  Indexing is available, but normally runs from React lifecycle, so unless you're using this with React indexes will be non-managed.
  "
  ([] (fulcro-app {}))
  ([{:keys [props-middleware
            batching-enabled
            global-eql-transform
            global-error-action
            default-result-action!
            core-render!
            optimized-render!
            batch-notifications
            render-root!
            hydrate-root!
            unmount-root!
            submit-transaction!
            abort-transaction!
            render-middleware
            before-render
            initial-db
            client-will-mount
            client-did-mount
            remote-error?
            remotes
            query-transform-default
            load-marker-default
            load-mutation
            root-class
            shared
            external-config
            refresh-component!
            shared-fn] :as options}]
   (let [tx! (or submit-transaction! txn/default-tx!)]
     (map->FulcroApp
       {:com.fulcrologic.fulcro.application/id           (tempid/uuid)
        :com.fulcrologic.fulcro.application/state-atom   (atom (or initial-db {}))
        :com.fulcrologic.fulcro.application/config       {:load-marker-default     load-marker-default
                                                          :batching-enabled        (or batching-enabled {})
                                                          :client-did-mount        (or client-did-mount (:started-callback options))
                                                          :client-will-mount       client-will-mount
                                                          :external-config         external-config
                                                          :query-transform-default query-transform-default
                                                          :load-mutation           load-mutation}
        :com.fulcrologic.fulcro.application/algorithms   (cond-> {:com.fulcrologic.fulcro.algorithm/tx!                    tx!
                                                                  :com.fulcrologic.fulcro.algorithm/vtx-render             (volatile! nil)
                                                                  :com.fulcrologic.fulcro.algorithm/abort!                 (or abort-transaction! txn/abort!)
                                                                  :com.fulcrologic.fulcro.algorithm/batch-notifications    batch-notifications
                                                                  :com.fulcrologic.fulcro.algorithm/core-render!           (or core-render! identity)
                                                                  :com.fulcrologic.fulcro.algorithm/optimized-render!      (or optimized-render! identity)
                                                                  :com.fulcrologic.fulcro.algorithm/initialize-state!      initialize-state!
                                                                  :com.fulcrologic.fulcro.algorithm/shared-fn              shared-fn
                                                                  :com.fulcrologic.fulcro.algorithm/render-root!           render-root!
                                                                  :com.fulcrologic.fulcro.algorithm/hydrate-root!          hydrate-root!
                                                                  :com.fulcrologic.fulcro.algorithm/unmount-root!          unmount-root!
                                                                  :com.fulcrologic.fulcro.algorithm/refresh-component!     refresh-component!
                                                                  :com.fulcrologic.fulcro.algorithm/render!                render!
                                                                  :com.fulcrologic.fulcro.algorithm/remote-error?          (or remote-error? default-remote-error?)
                                                                  :com.fulcrologic.fulcro.algorithm/global-error-action    global-error-action
                                                                  :com.fulcrologic.fulcro.algorithm/merge*                 merge/merge*
                                                                  :com.fulcrologic.fulcro.algorithm/default-result-action! (or default-result-action! mut/default-result-action!)
                                                                  :com.fulcrologic.fulcro.algorithm/global-eql-transform   (or global-eql-transform default-global-eql-transform)
                                                                  :com.fulcrologic.fulcro.algorithm/index-root!            indexing/index-root!
                                                                  :com.fulcrologic.fulcro.algorithm/index-component!       indexing/index-component!
                                                                  :com.fulcrologic.fulcro.algorithm/drop-component!        indexing/drop-component!
                                                                  :com.fulcrologic.fulcro.algorithm/props-middleware       props-middleware
                                                                  :com.fulcrologic.fulcro.algorithm/render-middleware      render-middleware
                                                                  :com.fulcrologic.fulcro.algorithm/schedule-render!       schedule-render!}
                                                           (fn? before-render) (assoc :com.fulcrologic.fulcro.algorithm/before-render before-render))
        :com.fulcrologic.fulcro.application/runtime-atom (atom
                                                           {:com.fulcrologic.fulcro.application/app-root            nil
                                                            :com.fulcrologic.fulcro.application/mount-node          nil
                                                            :com.fulcrologic.fulcro.application/root-class          root-class
                                                            :com.fulcrologic.fulcro.application/root-factory        nil
                                                            :com.fulcrologic.fulcro.application/basis-t             1
                                                            :com.fulcrologic.fulcro.application/last-rendered-state {}

                                                            :com.fulcrologic.fulcro.application/static-shared-props shared
                                                            :com.fulcrologic.fulcro.application/shared-props        {}

                                                            :com.fulcrologic.fulcro.application/remotes             (or remotes
                                                                                                                      {:remote {:transmit! (fn [{::txn/keys [result-handler]}]
                                                                                                                                             (log/fatal "Remote requested, but no remote defined.")
                                                                                                                                             (result-handler {:status-code 418 :body {}}))}})
                                                            :com.fulcrologic.fulcro.application/indexes             {:ident->components {}}
                                                            :com.fulcrologic.fulcro.application/mutate              mut/mutate
                                                            :com.fulcrologic.fulcro.application/render-listeners    (cond-> {}
                                                                                                                      (= tx! txn/default-tx!) (assoc ::txn/after-render txn/application-rendered!))
                                                            ::txn/activation-scheduled?                             false
                                                            ::txn/queue-processing-scheduled?                       false
                                                            ::txn/sends-scheduled?                                  false
                                                            ::txn/submission-queue                                  []
                                                            ::txn/active-queue                                      []
                                                            ::txn/send-queues                                       {}})}))))

(defn abort!
  "Attempt to abort the send queue entries with the given abort ID.

  NOTE: This can be redefined on an application. If you change your transaction processing routing, then the built-in
  version will not work, and this docstring does not apply.

  Will notify any aborted operations (e.g. result-handler
  will be invoked, remote-error? will be used to decide if you consider that an error, etc.).
  The result map from an abort will include `{::txn/aborted? true}`, but will not include `:status-code` or `:body`.

  This function affects both started and non-started items in the send queues, but will not affect submissions that have not yet
  made it to the network processing layer (things still in top-level transaction submission queue).

  So the sequence of calls:

  ```
  (comp/transact! this `[(f)] {:abort-id :a})
  (app/abort! this :a)
  ```

  will cancel anything active with abort id `:a`, but since you've held the thread the entire time the submission of
  mutation `(f)` is still on the submission queue and will not be aborted.

  - `app-ish`: Anything that can be coerced to an app with comp/any->app.
  - `abort-id`: The abort ID of the operations to be aborted.
  "
  [app-ish abort-id]
  (let [app (comp/any->app app-ish)]
    (when-let [abort! (ah/app-algorithm app :abort!)]
      (abort! app abort-id))
    nil))
