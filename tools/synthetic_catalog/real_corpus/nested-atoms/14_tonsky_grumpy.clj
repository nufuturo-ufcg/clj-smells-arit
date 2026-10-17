(ns grumpy.core.jobs
  (:require
   [clojure.java.shell :as shell]
   [clojure.string :as str]
   [grumpy.core.coll :as coll]
   [grumpy.core.log :as log]))

(defonce **agents (atom {}))

(defonce ^:dynamic current-agent-id nil)

(defn linearize-async [**agents agent-id f]
  (cond
    (nil? current-agent-id)
    (let [*a   (-> (swap! **agents coll/assoc-new agent-id (agent nil))
                 (get agent-id))
          *res (promise)]
      (send *a
        (fn [_]
          (try
            (binding [current-agent-id agent-id]
              (deliver *res [:success (f)]))
            nil
            (catch Throwable t
              (deliver *res [:throwable t])))))
      *res)

    (= current-agent-id agent-id)
    (deliver (promise) [:success (f)])

    (not= current-agent-id agent-id)
    (throw (IllegalStateException. (str "Trying to linearize " agent-id " while already in " current-agent-id)))))

(defn block [*promise timeout]
  (let [[status value] (deref *promise timeout [:timeout])]
    (case status
      :success   value
      :throwable (throw value)
      :timeout   (throw (java.util.concurrent.TimeoutException.)))))

(defmacro linearize [agent-id & body]
  `(-> (jobs/linearize-async **agents ~agent-id (fn [] ~@body))
     (block 60000)))
