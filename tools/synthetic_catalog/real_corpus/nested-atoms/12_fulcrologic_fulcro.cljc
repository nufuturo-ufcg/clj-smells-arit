(ns com.fulcrologic.fulcro.headless.timers
  "Timer control for deterministic timeout testing.

   This namespace provides utilities for mocking and controlling timers
   in Fulcro tests. This is particularly useful for testing:

   - UI State Machine timeouts
   - Dynamic routing delay/error timers
   - Debounced operations
   - Any code that uses `com.fulcrologic.fulcro.algorithms.scheduling/defer`

   Example:
   ```clojure
   (with-mock-timers
     (uism/begin! app LoginMachine ::login {:actor/form LoginForm} {})
     (uism/trigger! app ::login :submit {:username \"test\"})

     ;; Advance time to trigger timeout
     (advance-time! 5001)

     ;; Timer callbacks have now fired
     (is (= :error (uism/get-active-state app ::login))))
   ```"
  (:require
    [com.fulcrologic.fulcro.algorithms.scheduling :as sched]))

(def ^:dynamic *mock-time*
  "When bound, holds the current mock time in milliseconds."
  nil)

(def ^:dynamic *pending-timers*
  "When bound, holds an atom containing pending timer entries.
   Each entry: {:id unique-id :fire-at ms :callback fn :active volatile}"
  nil)

(def ^:dynamic *timer-id-counter*
  "Counter for generating unique timer IDs."
  nil)

(defn- next-timer-id []
  (swap! *timer-id-counter* inc))

(defn ^:no-doc mock-defer
  "Mock implementation of sched/defer that queues timers for manual control.
   Internal use - not part of public API."
  [f tm]
  (let [active   (volatile! true)
        timer-id (next-timer-id)
        fire-at  (+ @*mock-time* tm)
        cancel   (fn []
                   (vreset! active false)
                   (swap! *pending-timers*
                     (fn [timers] (filterv #(not= timer-id (:id %)) timers))))]
    (swap! *pending-timers* conj {:id       timer-id
                                  :fire-at  fire-at
                                  :callback f
                                  :active   active
                                  :delay-ms tm})
    cancel))
