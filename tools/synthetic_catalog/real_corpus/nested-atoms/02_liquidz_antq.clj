(ns ^:no-doc antq.report.table
  (:require
   [antq.log :as log]
   [antq.report :as report]
   [antq.util.dep :as u.dep]
   [antq.util.file :as u.file]
   [antq.util.report :as u.report]
   [antq.util.ver :as u.ver]
   [clojure.set :as set]
   [clojure.string :as str]))

(def ^:private progress (atom nil))

(defmethod report/init-progress "table"
  [deps _options]
  (reset! progress {:total-count (count deps)
                    :count-atom (atom 0)}))
