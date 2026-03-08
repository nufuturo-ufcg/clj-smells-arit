(ns misuse-of-channel-closing-semantics
  (:require [clojure.core.async :as a]))

(def my-chan (a/chan 1))

;; Sentinel :done — producer (put!)
(a/go
  (a/put! my-chan :done))

;; Sentinel :done — consumer (comparison)
(a/go
  (loop []
    (when-let [event (a/<! my-chan)]
      (when (not= event :done)
        (prn "Processing" event)
        (recur)))))

;; Sentinel :EOF
(a/go (a/put! my-chan :EOF))
(when (not= x :EOF) 1)

;; Sentinel :end
(a/go (a/put! my-chan :end))
(when (not= event :end) 2)

;; Sentinel ::end
(a/go (a/put! my-chan ::end))
(when (not= event ::end) 3)

;; Sentinel :eof
(a/go (a/put! my-chan :eof))
(when (not= event :eof) 4)

;; Sentinel :stream/done (namespaced)
(a/go (a/put! my-chan :stream/done))
(when (not= event :stream/done) 5)

;; Sentinel with >! (parking put)
(a/go (a/>! my-chan :done))

;; Sentinel with >!! (blocking put)
(a/thread (a/>!! my-chan :EOF))

;; Comparação com forma de take — deve ser reportada (sentinel em canal)
(when (= :done (a/<! my-chan)) (prn "channel closed"))
(when (not= (a/<! my-chan) :end) 1)

;; More sentinels (stem-based): :close, :synced, :return, :break, :hb-terminating
(a/go (a/put! my-chan :close))
(a/go (a/put! my-chan :synced))
(a/go (a/>! my-chan :return))
(a/thread (a/>!! my-chan :break))
(a/go (a/put! my-chan :hb-terminating))

;; Should NOT be reported: arbitrary keyword (no stem match)
(a/go (a/put! my-chan :foo))

;; --- Correct pattern (should not be reported): close! + when-let with nil ---
(def ok-chan (a/chan 1))
(a/go
  (a/>! ok-chan :some-event)
  (a/close! ok-chan))
(a/go
  (loop []
    (when-let [event (a/<! ok-chan)]
      (prn "Processing" event)
      (recur))))

;; --- Should not be reported: :done is alts!! default return, not channel sentinel ---
(defn drain-channels
  "Removes all messages from the given channels. Will not block."
  [channels]
  (when channels
    (loop []
      (when-not (= :done (first (a/alts!! (vec channels) :default :done)))
        (recur)))))
