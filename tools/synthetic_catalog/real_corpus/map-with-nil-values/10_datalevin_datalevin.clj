(ns ^:no-doc datalevin.dump
  "dump, load and re-index database"
  (:refer-clojure :exclude [load sync])
  (:require
   [clojure.string :as str]
   [clojure.pprint :as p]
   [clojure.edn :as edn]
   [taoensso.nippy :as nippy]
   [datalevin.util :as u]
   [datalevin.constants :as c]
   [datalevin.conn :as conn]
   [datalevin.db :as db]
   [datalevin.datom :as dd]
   [datalevin.lmdb :as l]
   [datalevin.interface :as i :refer [dir]])
  (:import
   [datalevin.db DB]
   [datalevin.datom Datom]
   [datalevin.storage Store]
   [datalevin.remote DatalogStore]
   [java.io PushbackReader FileOutputStream FileInputStream DataOutputStream
    DataInputStream IOException]))

(def ^:private nippy-meta-protocol-key
  :taoensso.nippy/meta-protocol-key)

(def ^:private legacy-ha-nil-sentinel-keys
  [:ha-mode
   :ha-control-plane
   :ha-members
   :ha-fencing-hook
   :ha-clock-skew-hook
   :ha-membership-hash])

(defn- normalize-legacy-ha-nil-sentinels
  [opts]
  (reduce
    (fn [m k]
      (if (= nippy-meta-protocol-key (get m k))
        (assoc m k nil)
        m))
    (or opts {})
    legacy-ha-nil-sentinel-keys))
