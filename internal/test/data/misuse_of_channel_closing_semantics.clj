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
