(ns metabase.app-db.core
  "API namespace for the application database (app-db).

  It has a few different functions you might need:
  - A connectible for the app-db [[app-db]] and [[data-source]]
  - Information about the app-db: [[db-is-set-up?]], [[db-type]], [[quoting-style]]
  - a few other random functions that have built up for different purposes.

  Namespaces outside of src/metabase/app_db/ should not use any metabase.app-db.* namespace but use this api namespace."
  (:refer-clojure :exclude [compile])
  (:require
   [clojure.core.memoize :as memoize]
   [metabase.app-db.connection :as mdb.connection]
   [metabase.app-db.connection-pool-setup :as mdb.connection-pool-setup]
   [metabase.app-db.data-source :as mdb.data-source]
   [metabase.app-db.encryption :as mdb.encryption]
   [metabase.app-db.env :as mdb.env]
   [metabase.app-db.format]
   [metabase.app-db.h2 :as mdb.h2]
   [metabase.app-db.jdbc-protocols :as mdb.jdbc-protocols]
   [metabase.app-db.liquibase :as liquibase]
   [metabase.app-db.query]
   [metabase.app-db.query-cancelation]
   [metabase.app-db.setup :as mdb.setup]
   [metabase.app-db.spec :as mdb.spec]
   [metabase.config.core :as config]
   [potemkin :as p]))

(defn memoize-for-application-db
  "Like [[clojure.core/memoize]], but only memoizes for the current application database; memoized values will be
  ignored if the app DB is dynamically rebound. For TTL memoization with [[clojure.core.memoize]], set
  `:clojure.core.memoize/args-fn` instead; see [[metabase.driver.util/database->driver*]] for an example of how to do
  this."
  [f]
  (memoize/memo
   (vary-meta
    (fn [& args] (apply f args))
    assoc ::memoize/args-fn (fn [args] (cons (unique-identifier) args)))))

(defn increment-app-db-unique-indentifier!
  "Increment the [[unique-identifier]] for the Metabase application DB. This effectively flushes all caches using it as
  a key (including things using [[mdb/memoize-for-application-db]]) such as the Settings cache. Should only be used
  for testing. Not general purpose."
  []
  (assert (or (not config/is-prod?)
              (config/config-bool :mb-enable-test-endpoints)))
  (alter-var-root #'mdb.connection/*application-db* assoc :id (swap! mdb.connection/application-db-counter inc)))
