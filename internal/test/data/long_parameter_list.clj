(ns long-parameter-list)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Function with 10 parameters
(defn register-new-user
  [username password email phone age gender location interests newsletter-opt-in referred-by]
  {:username username
   :password password
   :email email
   :phone phone
   :age age
   :gender gender
   :location location
   :interests interests
   :newsletter newsletter-opt-in
   :referral referred-by})

;; Example 2: Function with 11 parameters
(defn schedule-meeting
  [title description date time duration participants organizer room-id reminders notes calendar-id]
  {:title title
   :description description
   :datetime {:date date :time time}
   :duration duration
   :participants participants
   :organizer organizer
   :room room-id
   :reminders reminders
   :notes notes
   :calendar calendar-id})

;; Example 3: Function with 13 parameters
(defn configure-server
  [host port ssl? db-host db-port db-name username password
   timeout retries region availability-zone monitoring-enabled]
  {:server {:host host :port port :ssl ssl?}
   :database {:host db-host :port db-port :name db-name}
   :auth {:user username :pass password}
   :network {:timeout timeout :retries retries}
   :location {:region region :zone availability-zone}
   :monitoring monitoring-enabled})


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========


;; Example 4: Function with 1 parameter that contains 10 primitive fields inside
(defn process-user-registration
  [{:keys [username password email phone age gender location
           interests newsletter-opt-in referred-by] :as user-data}]
  {:account {:user username
             :pass password
             :email email
             :phone phone}
   :profile {:age age
             :gender gender
             :location location
             :interests interests}
   :settings {:newsletter newsletter-opt-in
              :referral referred-by}
   :raw user-data})

;; Example 5: Function with 9 parameters (limit case)
(defn generate-user-report
  [user-id start-date end-date include-activity? include-purchases?
   include-location? format language send-email?]
  {:user user-id
   :range [start-date end-date]
   :options {:activity include-activity?
             :purchases include-purchases?
             :location include-location?}
   :output {:format format :language language :notify send-email?}})

;; Example 6: Function with 3 parameters
(defn send-notification
  [user-id message timestamp]
  {:user user-id
   :text message
   :time timestamp})

;; Example 7: Function with 6 parameters
(defn calculate-order-total
  [item-prices tax discount shipping-fee coupon-code membership-discount]
  (let [subtotal (reduce + item-prices)
        after-discount (- subtotal discount membership-discount)
        after-coupon (* after-discount coupon-code)
        total (+ after-coupon tax shipping-fee)]
    total))