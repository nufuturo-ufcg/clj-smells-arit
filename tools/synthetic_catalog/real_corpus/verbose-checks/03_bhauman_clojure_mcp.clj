(ns clojure-mcp.nrepl-launcher
  "Handles automatic nREPL server startup and port discovery."
  (:require [clojure-mcp.utils.valid-paths :as valid-paths]
            [clojure.edn :as edn]
            [clojure.java.io :as io]
            [clojure.java.process :as process]
            [clojure.string :as str]
            [nrepl.version :as nrepl-version]
            [taoensso.timbre :as log])
  (:import [java.io BufferedReader IOException]
           [java.net InetSocketAddress Socket]
           [java.util.concurrent TimeUnit]))

(defn parse-port-from-output
  "Parse nREPL port number from process output.
   Looks for common patterns like 'port 12345', ':12345', 'nREPL server started on port 12345'.
   Returns the port number as an integer or nil if not found."
  [output]
  (when output
    (let [patterns [#"(?i)\bport\s{1,10}(\d{4,5})\b" ; port 12345
                    #":\s?(\d{4,5})\b" ; :12345 or : 12345 (4-5 digit ports)
                    #"(?i)Started on\s{1,10}(\d{4,5})\b" ; Started on 12345
                    #"(?i)Listening on\s{1,10}(\d{4,5})\b"] ; Listening on 12345
          line (str/trim output)]
      (some (fn [pattern]
              (when-let [match (re-find pattern line)]
                (when-let [port-str (second match)]
                  (try
                    (let [port (Integer/parseInt port-str)]
                      (when (<= 1024 port 65535) ; Valid port range
                        port))
                    (catch NumberFormatException _ nil)))))
            patterns))))

(defn wait-for-port
  "Monitor process stdout for port information.
   Returns port number when found or nil on timeout.
   Uses BufferedReader.ready() for proper non-blocking character reads."
  [^Process process timeout-ms parse-port?]
  (if-not parse-port?
    (do
      (log/info "Port parsing disabled, skipping port discovery")
      nil)
    (let [^BufferedReader reader (io/reader (.getInputStream process))
          start-time (System/currentTimeMillis)
          buffer (StringBuilder.)
          char-buffer (char-array 1024)]
      (try
        (loop []
          (when (< (- (System/currentTimeMillis) start-time) timeout-ms)
            ;; Check if character data is ready to read (non-blocking)
            (if (.ready reader)
              ;; Read available characters into buffer
              (let [chars-read (.read reader char-buffer)]
                (when (> chars-read 0)
                  (.append buffer char-buffer 0 chars-read)
                  (log/debug "Read" chars-read "chars, buffer now contains:" (str buffer))
                  ;; Check accumulated buffer for port
                  (if-let [port (parse-port-from-output (str buffer))]
                    (do
                      (log/info (str "Discovered nREPL port: " port))
                      port)
                    ;; No port found yet, continue accumulating
                    (do
                      (Thread/sleep 10)
                      (recur)))))
              ;; No character data ready, check if process is still alive
              (if (.isAlive process)
                (do
                  (Thread/sleep 100)
                  (recur))
                ;; Process exited, check final buffer content
                (if-let [port (parse-port-from-output (str buffer))]
                  (do
                    (log/info (str "Discovered nREPL port from final buffer: " port))
                    port)
                  (do
                    (log/warn "Process exited before port was discovered")
                    nil))))))
        (catch Exception e
          (log/error e "Error reading process output")
          nil)
        (finally
          (try
            (.close reader)
            (catch Exception _)))))))
