(ns shadow.grove
  "grove - a small wood or forested area (ie. trees)
   a mini re-frame/fulcro hybrid. re-frame event styles + somewhat normalized db"
  (:require-macros [shadow.grove])
  (:require
    [clojure.core.protocols :as cp]
    [goog.async.nextTick]
    [shadow.arborist.protocols :as ap]
    [shadow.arborist.common :as common]
    [shadow.arborist.fragments] ;; util macro references this
    [shadow.arborist :as sa]
    [shadow.arborist.collections :as sc]
    [shadow.grove.protocols :as gp]
    [shadow.grove.runtime :as rt]
    [shadow.grove.components :as comp]
    [shadow.grove.ui.util :as util]
    [shadow.grove.ui.suspense :as suspense]
    [shadow.grove.ui.atoms :as atoms]
    [shadow.grove.ui.portal :as portal]
    [shadow.arborist.attributes :as a]
    [shadow.grove.kv :as kv]
    [shadow.grove.impl :as impl]
    [shadow.grove.trace :as trace]
    [shadow.css] ;; used in macro ns
    ))

(defn run-tx! [runtime-ref tx]
  (assert (rt/ref? runtime-ref) "expected runtime ref?")
  (let [{::keys [scheduler]} @runtime-ref]
    (let [tx (cond
               (fn? tx) (with-meta {:e ::fn!} {::tx tx})
               (and (map? tx) (keyword? (:e tx))) tx
               :else (throw (js/Error. "run-tx! only accepts functions or maps as tx argument")))]

      (impl/process-event runtime-ref tx nil nil)
      )))

(deftype RootScheduler [^:mutable update-pending? work-set]
  gp/IScheduleWork
  (schedule-work! [this work-task trigger]
    (.add work-set work-task)

    (when-not update-pending?
      (set! update-pending? true)
      (rt/microtask #(.microtask-start this trigger))))

  (unschedule! [this work-task]
    (.delete work-set work-task))

  (did-suspend! [this target])
  (did-finish! [this target])

  Object
  (microtask-start [this trigger]
    (let [t (trace/run-microtask this trigger)]
      (try
        (.process-work! this trigger)
        (finally
          (trace/run-microtask-done this trigger t))))

    js/undefined)

  (process-work! [this trigger]
    (let [t (trace/run-work this trigger)]
      (try
        (let [iter (.values work-set)]
          (loop []
            (let [current (.next iter)]
              (when (not ^boolean (.-done current))
                ;; kick off work from scheduler root, going down through components and their schedulers
                (gp/work! ^not-native (.-value current))

                (recur)))))

        js/undefined

        (finally
          (trace/run-work-done this trigger t)
          (set! update-pending? false))))

    js/undefined))

(defn make-kv-navigable [init-data]
  ;; a bit of nested madness. trying to keep all nav related things only active
  ;; when navigating from :shadow.grove.kv itself. nav from a table can't follow references
  ;; if it does not have a kv container map itself. nav should only
  ;; use the data from that kv map instance and not get a new snapshot from data-ref
  ;; since that can lead to inconsistencies, e.g. something already being removed.
  (with-meta init-data
    {`cp/nav
     (fn [kv table-id table]
       ;; nav in table, kv only contains other tables
       (vary-meta table assoc
         `cp/nav
         (fn [table entry-key v]
           ;; only navigate further on maps for now
           (if-not (map? v)
             v
             ;; nav in entity map
             (vary-meta v assoc
               `cp/nav
               (fn [entry k v]
                 ;; e.g. :runtime-id nil, can't navigate further if value was nil
                 (if (nil? v)
                   v
                   (let [fk-reference (some-> (meta table) ::kv/config :attrs (get k) :references)]
                     (if-not fk-reference
                       v
                       (let [table (cp/nav kv fk-reference (get kv fk-reference))]
                         (cp/nav table v (get table v)))
                       )))))))))}))

(defn- prepare [app-id]
  (when (get @rt/known-runtimes-ref app-id)
    (throw
      (ex-info
        (str "app " app-id " already registered!")
        {:app-id app-id})))

  (let [root-scheduler
        (RootScheduler. false (js/Set.))

        rt-ref
        (atom
          {::runtime true
           ::roots #{}
           ::scheduler root-scheduler
           ::app-id app-id
           ::kv (make-kv-navigable {})
           ::event-config {}
           ::event-interceptors [impl/kv-interceptor]
           ::event-error-handler
           (fn [env ev origin ex]
             ;; re-throwing is total shit in JS, so instead just use console
             ;; if users want to handle this differently they can replace this fn
             (js/console.error "--- FAILED TO PROCESS EVENT ---" ev)
             (js/console.error ex)
             ::failed!)
           ::fx-config {}
           ::timeouts {}
           ::env-init []})]

    (swap! rt-ref update ::fx-config merge
      {::timeout
       (fn [env {:keys [timeout-id timeout ev]}]
         (let [tid (js/setTimeout
                     (fn []
                       (swap! rt-ref update ::timeouts dissoc timeout-id)
                       (run-tx! rt-ref ev))
                     timeout)]

           (swap! rt-ref update ::timeouts assoc timeout-id tid)))

       ::timeout-clear
       (fn [env {:keys [timeout-id]}]
         (when-some [tid (get-in @rt-ref [::timeouts timeout-id])]
           (js/clearTimeout tid)
           (swap! rt-ref update ::timeouts dissoc timeout-id)
           ))})

    (swap! rt/known-runtimes-ref assoc app-id rt-ref)

    rt-ref))

(def kv-interceptor impl/kv-interceptor)
