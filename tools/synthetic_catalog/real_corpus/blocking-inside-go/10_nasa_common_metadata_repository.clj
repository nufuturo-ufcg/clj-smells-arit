(ns cmr.common.jobs
  "Defines a job scheduler that wraps quartz for defining a job."
  (:require
   [clj-time.core :as t]
   [cmr.common.config :as config :refer [defconfig]]
   [cmr.common.lifecycle :as l]
   [cmr.common.log :as log :refer (debug info warn error)]
   [cmr.common.services.errors :as errors]
   [clojure.core.async :as async]
   ;; quartzite dependencies
   [clojurewerkz.quartzite.conversion :as qc]
   [clojurewerkz.quartzite.jobs :as qj]
   [clojurewerkz.quartzite.schedule.calendar-interval :as qcal]
   [clojurewerkz.quartzite.schedule.cron :as qcron]
   [clojurewerkz.quartzite.scheduler :as qs]
   [clojurewerkz.quartzite.stateful :as qst]
   [clojurewerkz.quartzite.triggers :as qt])
  (:import
   (org.quartz.impl StdScheduler)))

(defconfig default-job-start-delay
  "The start delay of the job in seconds."
  {:default 5
   :type Long})

(defmulti create-trigger
  "Creates a trigger for the given job."
  (fn [_job-key job]
    (cond
      (:interval job) :interval
      (:daily-at-hour-and-minute job) :daily-at-hour-and-minute
      :else :default)))

(defmethod create-trigger :default
  [_job-key _job]
  (errors/internal-error!
    (str "Job could not be scheduled. One of :interval or "
         ":daily-at-hour-and-minute should be set.")))

(defmethod create-trigger :interval
  [job-key {:keys [start-delay interval]}]
  (info (format "Scheduling job %s with interval %s" job-key interval))
  (qt/build
    (qt/with-identity (qt/key (str job-key ".trigger")))
    ;; Set start delay
    (qt/start-at (-> (or start-delay (default-job-start-delay))
                     t/seconds
                     t/from-now))
    (qt/with-schedule
      (qcal/schedule (qcal/with-interval-in-seconds interval)))))

(defmethod create-trigger :daily-at-hour-and-minute
  [job-key {[hour minute] :daily-at-hour-and-minute}]
  (info (format "Scheduling job %s daily at %s:%s" job-key hour minute))
  (qt/build
    (qt/with-identity (qt/key (str job-key ".trigger")))
    (qt/with-schedule (qcron/daily-at-hour-and-minute hour minute))))

(defn- try-to-schedule-job
  "Attempts to schedule a job. Swallows the exception if there is an error.
  Returns true if the job was successfully scheduled and false otherwise."
  [scheduler job-key quartz-job trigger]
  (try
    ;; We delete existing jobs and recreate them
    (when (qs/get-job scheduler job-key)
      (qs/delete-job scheduler (qj/key job-key)))
    (qs/schedule scheduler quartz-job trigger)
    true
    (catch Exception e
      ;; This normally happens locally if the system is not configured to run jobs or have some
      ;; other strange deployment, that is why the bomb emoji is used in the log message.
      (warn (format "💣 - Error scheduling job [%s]: %s" job-key (.getMessage e)))
      (debug e (format "Full exception for job [%s]" job-key))
      false)))

(defn- schedule-job
  "Schedules a quartzite job (stopping existing job first). This can fail due
  to race conditions with other nodes trying to schedule the job at the same
  time. We will retry up to 3 times to schedule the job."
  [scheduler system-holder-var-name job]
  (let [{:keys [^Class job-type job-key]} job
        job-key (or job-key (str (.getSimpleName job-type) ".job"))
        quartz-job (qj/build
                    (qj/of-type job-type)
                    (qj/using-job-data
                     {"system-holder-var-name" system-holder-var-name})
                    (qj/with-identity (qj/key job-key)))
        trigger (create-trigger job-key job)]
    (async/go-loop [max-tries 3]
      (when-not (try-to-schedule-job scheduler job-key quartz-job trigger)
        (if (pos? max-tries)
          (do
            (warn (format "Failed to schedule job [%s]. Retrying." job-key))
            ;; Random sleep time to make it less likely that two nodes try to
            ;; recreate the job at the same time. Sleeps between 0.5 seconds
            ;; and 3 seconds.
            (Thread/sleep (+ 500 (rand-int 2500)))
            (recur (dec max-tries)))
          (warn
           (format
            "All retries to schedule job [%s] failed." job-key)))))))
