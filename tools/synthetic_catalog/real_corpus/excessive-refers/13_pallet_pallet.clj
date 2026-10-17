(ns pallet.actions
  "Pallet's action primitives."
  (:require
   [clj-schema.schema
    :refer [constraints def-map-schema map-schema optional-path sequence-of]]
   [clojure.java.io :as io]
   [clojure.set :refer [intersection]]
   [clojure.string :refer [trim]]
   [clojure.tools.logging :as logging]
   [pallet.action
    :refer [clj-action defaction enter-scope get-action-options leave-scope
            with-action-options]]
   [pallet.action-plan :refer [checked-script]]
   [pallet.actions-impl :refer :all]
   [pallet.argument :as argument :refer [delayed delayed-argument?]]
   [pallet.contracts :refer [any-value check-spec]]
   [pallet.core.file-upload :refer :all]
   [pallet.core.session :refer [session]]
   [pallet.crate :refer [admin-user packager phase-context role->nodes-map
                         target]]
   [pallet.node-value :refer [node-value]]
   [pallet.script.lib :as lib :refer [set-flag-value user-home]]
   [pallet.stevedore :as stevedore :refer [fragment with-source-line-comments]]
   [pallet.utils :refer [apply-map log-multiline tmpfile]])
  (:import clojure.lang.Keyword))
