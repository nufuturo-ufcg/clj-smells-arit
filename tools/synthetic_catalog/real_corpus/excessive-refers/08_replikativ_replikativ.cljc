(ns replikativ.stage
  "A stage allows to execute upstream operations of each CRDT and
  communicates them downstream to a peer through
  synchronous (blocking) operations."
  (:require [konserve.core :as k]
            [kabel.peer :refer [drain]]
            [replikativ.core :refer [wire]]
            [replikativ.connect :refer [connect]]
            [replikativ.protocols :refer [-downstream]]
            [replikativ.environ :refer [*id-fn* store-blob-trans-id store-blob-trans-value]]
            [replikativ.crdt.materialize :refer [key->crdt]]
            [replikativ.p2p.fetch :refer [fetch]]
            [kabel.middleware.block-detector :refer [block-detector]]
            #?(:clj [kabel.platform-log :refer [debug info warn]])
            #?(:clj [superv.async :refer [<? <<? go-try go-loop-try alt? put?
                                          go-for go-loop-super >?]])
            #?(:cljs [superv.async :refer [<? <<? go-try go-loop-try alt? put?
                                           go-for go-loop-super >?] :include-macros true])
            [hasch.core :refer [uuid]]
            [clojure.set :as set]
            #?(:clj [clojure.core.async :as async
                     :refer [<! >! timeout chan put! sub unsub pub close! alt! onto-chan]]
               :cljs [clojure.core.async :as async
                      :refer [<! >! timeout chan put! sub unsub pub close! onto-chan] :include-macros true]))
  #?(:cljs (:require-macros [replikativ.stage :refer [go-try-locked]]
                            [kabel.platform-log :refer [debug info warn]])))
