(ns datahike.query.execute
  "Execution engine for query plans.
   Supports fused scan+merge for entity groups, hash-probe value joins,
   anti-merge NOT, and direct-to-HashSet output."
  (:require
   [datahike.constants :refer [e0 tx0 emax txmax]]
   [datahike.datom :as datom :refer [datom]]
   [datahike.db.interface :as dbi]
   [datahike.db.utils :as dbu]
   [datahike.index.interface :as di]
   [datahike.query.analyze :as analyze]
   [datahike.query.plan :as plan]
   [datahike.query.relation :as rel]
   #?(:clj [datahike.index.secondary :as sec])
   #?(:clj [datahike.index.entity-set :as es])
   #?(:clj [datahike.query :as legacy])
   #?(:cljs [org.replikativ.persistent-sorted-set :as psset])
   #?(:cljs [org.replikativ.persistent-sorted-set.btset :as btset :refer [BTSet]])
   [datahike.db :as db #?@(:cljs [:refer [AsOfDB SinceDB HistoricalDB FilteredDB]])]
   [replikativ.logging :as log])
  #?(:cljs (:require-macros [datahike.query.execute :refer [scan-filter scan-filter-temporal emit-tuple check-cancel!]]))
  #?(:clj (:import [datahike.datom Datom]
                   [datahike.db AsOfDB SinceDB HistoricalDB FilteredDB]
                   [org.replikativ.persistent_sorted_set
                    PersistentSortedSet
                    PersistentSortedSet$ForwardCursor])))

#?(:clj (import 'datahike.query.relation.Relation))
