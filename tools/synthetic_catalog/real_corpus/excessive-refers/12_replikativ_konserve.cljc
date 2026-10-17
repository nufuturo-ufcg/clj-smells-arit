(ns konserve.tiered
  "Tiered store implementation with frontend and backend storage layers."
  (:refer-clojure :exclude [get get-in update update-in assoc assoc-in exists? dissoc keys])
  (:require [clojure.core.async :refer [go] :as async]
            [clojure.set :as set]
            [konserve.memory :as memory]
            [konserve.protocols :as protocols :refer [-exists? -get-meta -get-in -assoc-in
                                                      -update-in -dissoc -bget -bassoc
                                                      -keys -multi-get -multi-assoc -multi-dissoc -assoc-serializers
                                                      PEDNKeyValueStore PBinaryKeyValueStore
                                                      PKeyIterable PAssocSerializers PMultiKeySupport
                                                      PMultiKeyEDNValueStore]]
            [konserve.utils :refer [meta-update multi-key-capable? kv-keys invoke-write-hooks! #?(:clj async+sync) *default-sync-translation*]
             #?@(:cljs [:refer-macros [async+sync]])]
            [superv.async :refer [go-try- <?-]]
            [replikativ.logging :as log]))
