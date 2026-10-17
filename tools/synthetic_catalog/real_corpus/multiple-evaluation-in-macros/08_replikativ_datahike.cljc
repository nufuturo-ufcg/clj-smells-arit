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

(defn- val-eq?
  "Value equality that handles byte arrays (which don't support Clojure =)."
  [a b]
  #?(:clj (if (and (bytes? a) (bytes? b))
            (java.util.Arrays/equals ^bytes a ^bytes b)
            (= a b))
     :cljs (= a b)))

#?(:clj
   (defmacro ^:private merge-datom-match?
     "Inline predicate: does merge datom d match entity+attr+value+shared-var constraints?
      Expands in place for zero-overhead in hot loops. Type-hints d and scan-d internally."
     [d eid ra vg? vgv eq-v eq-tx scan-d merge-datoms]
     (let [d# (with-meta d {:tag 'datahike.datom.Datom})]
       `(and (== (.-e ~d#) ~eid) (= (.-a ~d#) ~ra)
             (or (not ~vg?) (val-eq? (.-v ~d#) ~vgv))
             (eq-ok? ~eq-v (.-v ~d#) ~d# ~scan-d ~merge-datoms)
             (eq-ok? ~eq-tx (datom/datom-tx ~d#) ~d# ~scan-d ~merge-datoms)))))
