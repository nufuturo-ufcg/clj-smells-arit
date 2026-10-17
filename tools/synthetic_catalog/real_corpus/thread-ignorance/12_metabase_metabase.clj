(ns metabase-enterprise.serialization.cmd
  (:require
   [clojure.java.io :as io]
   [clojure.string :as str]
   [metabase-enterprise.serialization.v2.extract :as v2.extract]
   [metabase-enterprise.serialization.v2.ingest :as v2.ingest]
   [metabase-enterprise.serialization.v2.load :as v2.load]
   [metabase-enterprise.serialization.v2.storage :as v2.storage]
   [metabase-enterprise.serialization.v2.storage.files :as v2.storage.files]
   [metabase.analytics.core :as analytics]
   [metabase.app-db.core :as mdb]
   [metabase.events.core :as events]
   [metabase.models.serialization :as serdes]
   [metabase.plugins.core :as plugins]
   [metabase.premium-features.core :as premium-features]
   [metabase.setup.core :as setup]
   [metabase.util :as u]
   [metabase.util.i18n :refer [trs]]
   [metabase.util.log :as log]
   [metabase.util.malli :as mu]
   [toucan2.core :as t2])
  (:import
   (clojure.lang ExceptionInfo)))

(defn- check-premium-token! []
  (premium-features/assert-has-feature :serialization (trs "Serialization")))

(mu/defn v2-load-internal!
  "SerDes v2 load entry point for internal users.

  `opts` are passed to [[v2.load/load-metabase]]."
  [path :- :string
   opts :- [:map
            [:continue-on-error {:optional true} [:maybe :boolean]]
            [:reindex? {:optional true} [:maybe :boolean]]]
   ;; Deliberately separate from the opts so it can't be set from the CLI.
   & {:keys [token-check?
             require-initialized-db?]
      :or   {token-check? true
             require-initialized-db? true}}]
  (plugins/load-plugins!)
  (mdb/setup-db! :create-sample-content? false)
  (when (and require-initialized-db? (not (setup/has-user-setup)))
    (throw (ex-info "You cannot `import` into an empty database. Please set up Metabase normally, then retry." {})))
  (when token-check?
    (check-premium-token!))
  ;; TODO This should be restored, but there's no manifest or other meta file written by v2 dumps.
  ;;(when-not (load/compatible? path)
  ;;  (log/warn "Dump was produced using a different version of Metabase. Things may break!"))
  (log/infof "Loading serialized Metabase files from %s" path)
  (u/prog1 (serdes/with-cache
             (v2.load/load-metabase! (v2.ingest/ingest-yaml path) opts))
    (events/publish-event! :event/serdes-load {})))

(mu/defn v2-load!
  "SerDes v2 load entry point.

   opts are passed to load-metabase"
  [path :- :string
   opts :- [:map
            [:continue-on-error {:optional true} [:maybe :boolean]]
            [:full-stacktrace {:optional true} [:maybe :boolean]]]]
  (let [timer    (u/start-timer)
        err      (atom nil)
        report   (try
                   (v2-load-internal! path opts :token-check? true)
                   (catch ExceptionInfo e
                     (reset! err e))
                   (catch Exception e
                     (reset! err e)))
        imported (into (sorted-set) (map (comp :model last)) (:seen report))]
    (analytics/track-event! :snowplow/serialization
                            {:event         :serialization
                             :direction     "import"
                             :source        "cli"
                             :duration_ms   (int (u/since-ms timer))
                             :models        (str/join "," imported)
                             :count         (if (contains? imported "Setting")
                                              (inc (count (remove #(= "Setting" (:model (first %))) (:seen report))))
                                              (count (:seen report)))
                             :error_count   (count (:errors report))
                             :success       (nil? @err)
                             :error_message (when @err
                                              (u/strip-error @err nil))})
    (when @err
      (if (:full-stacktrace opts)
        (log/error @err "Error during deserialization")
        (log/error (u/strip-error @err "Error during deserialization")))
      (throw (ex-info (ex-message @err) {:cmd/exit true})))
    imported))
