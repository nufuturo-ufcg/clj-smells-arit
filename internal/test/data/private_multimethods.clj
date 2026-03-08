(ns private-multimethod-smell)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: defmulti defined inside defn-
(defn- event-handler [event]
  (defmulti dispatch-event :type)

  (defmethod dispatch-event :create [e]
    (println "Create event"))

  (dispatch-event event))

;; Example 2: defmulti inside ^:private function
(defn ^:private payment-handler [payment]
  (defmulti process-payment :method)

  (defmethod process-payment :credit [p]
    (println "Credit payment"))

  (defmethod process-payment :debit [p]
    (println "Debit payment"))

  (process-payment payment))

;; Example 3: defmethod inside private function
(defmulti task-runner :task)

(defn- register-task-methods []
  (defmethod task-runner :build [t]
    (println "Building"))

  (defmethod task-runner :test [t]
    (println "Testing")))

;; Example 4: private multimethod (Analisar se faz sentido)
(defmulti ^:private route-dispatch :page)

(defmethod route-dispatch :home [_]
  :home-page)

(defmethod route-dispatch :about [_]
  :about-page)

;; Example 5: multimethod defined inside helper private function
(defn- internal-dispatcher [msg]
  (defmulti handle-message :kind)

  (defmethod handle-message :email [m]
    (println "Email"))

  (defmethod handle-message :sms [m]
    (println "SMS"))

  (handle-message msg))

;; Example 6: nested private function defining multimethod
(defn outer []
  (letfn [(inner []
            (defmulti process-item :type)

            (defmethod process-item :a [_] :A)
            (defmethod process-item :b [_] :B))]
    (inner)))

;; Example 7: private multimethod using metadata (Analisar se faz sentido)
(defmulti ^{:private true} internal-dispatch :kind)

(defmethod internal-dispatch :alpha [_]
  :alpha)

;; Example 8: private function wrapping multimethod definition
(defn ^:private setup-handler []
  (defmulti handle-event :type)

  (defmethod handle-event :login [_]
    (println "login"))

  (defmethod handle-event :logout [_]
    (println "logout")))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 9: public multimethod definition
(defmulti handle-event :type)

(defmethod handle-event :create [_]
  (println "Create"))

(defmethod handle-event :delete [_]
  (println "Delete"))

;; Example 10: private function calling multimethod
(defn- run-handler [event]
  (handle-event event))

;; Example 11: private helper without multimethod
(defn- helper [x]
  (+ x 1))

;; Example 12: multimethod defined globally and used later
(defmulti dispatch :kind)

(defmethod dispatch :a [_]
  :A)

(defmethod dispatch :b [_]
  :B)

(defn run-dispatch [x]
  (dispatch x))

;; Example 13: private function returning function (no multimethod)
(defn- build-handler []
  (fn [x] (* x 2)))

;; Example 14: polymorphism using case instead of multimethod
(defn process-event [event]
  (case (:type event)
    :create (println "create")
    :delete (println "delete")
    :unknown))

;; Example 15: multimethod with public visibility and external extension
(defmulti format-output :format)

(defmethod format-output :json [data]
  (str "json:" data))

(defmethod format-output :xml [data]
  (str "xml:" data))

;; Example 16: local function with letfn but no multimethod
(defn compute [x]
  (letfn [(double [y] (* y 2))]
    (double x)))

;; Example 17: multimethod defined outside, private function uses it
(defmulti render :type)

(defmethod render :text [_]
  "text")

(defn ^:private render-internal [x]
  (render x))

;; Example 18: protocol-based polymorphism instead of multimethod
(defprotocol Processor
  (process [this]))

(defrecord Task []
  Processor
  (process [_] :done))

;; Example 19: simple map-based dispatch
(def handlers
  {:create #(println "create")
   :delete #(println "delete")})

(defn run-handler-map [event]
  ((handlers (:type event)) event))

;; Example 20: higher-order function instead of multimethod
(defn make-processor [f]
  (fn [x] (f x)))
