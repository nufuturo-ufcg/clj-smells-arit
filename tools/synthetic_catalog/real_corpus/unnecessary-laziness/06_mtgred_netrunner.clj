(ns web.lobby
  (:require
    [clojure.core.async :refer [<! go timeout]]
    [cljc.java-time.instant :as inst]
    [cljc.java-time.duration :as duration]
    [cljc.java-time.temporal.chrono-unit :as chrono]
    [clojure.set :as set]
    [clojure.string :as str]
    [com.climate.claypoole :as cp]
    [crypto.password.bcrypt :as bcrypt]
    [game.core :as core]
    [game.utils :refer [server-card]]
    [jinteki.utils :refer [constructed-game? select-non-nil-keys side-from-str superuser? to?]]
    [jinteki.preconstructed :refer [all-matchups]]
    [jinteki.validator :as validator]
    [medley.core :refer [find-first]]
    [monger.collection :as mc]
    [time-literals.read-write :as read-write]
    [web.app-state :as app-state]
    [web.mongodb :as mongodb]
    [web.stats :as stats]
    [web.ws :as ws]
    [taoensso.encore :as enc]
    [taoensso.timbre :as timbre]))

(defonce pool-size (+ 2 (.availableProcessors (Runtime/getRuntime))))

(defonce game-pools
  (letfn [(new-pool [x]
            {:pool (cp/threadpool 1 {:name (str "game-thread-" x)})
             :occupants (atom #{})})] ;; note that occupants is just for load balancing purposes
    (vec (map new-pool (range pool-size)))))
