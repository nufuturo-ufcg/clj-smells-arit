;; Exemplos de Redundant Do Block
;; Este arquivo contém exemplos de blocos 'do' redundantes que são desnecessários
;; porque suas formas pai já fornecem um 'do' implícito.

(ns examples.smells.redundant-do-block)

;; Exemplo 1: 'do' redundante em 'when'
(defn process-when-valid [data]
  (when (valid? data)
    (do
      (log "Processing data")
      (transform data)
      (save-result data))))

;; Exemplo 2: 'do' redundante em 'let'
(defn calculate-total [items]
  (let [prices (map :price items)
        tax-rate 0.1]
    (do
      (println "Calculating total")
      (+ (reduce + prices) (* (reduce + prices) tax-rate)))))

;; Exemplo 3: 'do' redundante em 'if'
(defn handle-user-input [input]
  (if (valid-input? input)
    (do
      (log-input input)
      (process-input input))
    (do
      (log-error "Invalid input")
      (show-error-message))))

;; Exemplo 4: 'do' redundante em 'defn'
(defn initialize-system []
  (do
    (load-config)
    (setup-database)
    (start-services)))

;; Exemplo 5: 'do' redundante em 'fn'
(def process-data
  (fn [data]
    (do
      (validate data)
      (transform data)
      (persist data))))

;; Exemplo 6: 'do' redundante em 'when-let'
(defn process-user [user-id]
  (when-let [user (find-user user-id)]
    (do
      (update-last-seen user)
      (send-notification user)
      user)))

;; Exemplo 7: 'do' redundante em 'try'
(defn safe-operation [data]
  (try
    (do
      (validate-data data)
      (risky-operation data))
    (catch Exception e
      (do
        (log-error e)
        (handle-error e)))))

;; Exemplo 8: 'do' redundante em 'cond'
(defn categorize-age [age]
  (cond
    (< age 13) (do
                 (log "Child category")
                 :child)
    (< age 20) (do
                 (log "Teen category")
                 :teen)
    :else (do
            (log "Adult category")
            :adult)))

;; Exemplo 9: 'do' redundante em 'case'
(defn handle-status [status]
  (case status
    :pending (do
               (log "Status is pending")
               (schedule-retry))
    :success (do
               (log "Operation successful")
               (cleanup-resources))
    :error (do
             (log "Operation failed")
             (alert-admin))))

;; Exemplo 10: 'do' redundante em 'finally'
(defn cleanup-operation []
  (try
    (perform-operation)
    (finally
      (do
        (close-connections)
        (cleanup-temp-files)
        (log "Cleanup completed")))))

;; Exemplo 11: 'do' redundante vazio
(defn empty-do-example []
  (when true
    (do)))

;; Exemplo 12: 'do' redundante com uma única expressão
(defn single-expression-do []
  (when (ready?)
    (do
      (start-process))))

;; Exemplo 13: 'do' redundante em multi-arity function
(defn multi-arity-example
  ([x]
   (do
     (println "Single arg")
     (* x 2)))
  ([x y]
   (do
     (println "Two args")
     (+ x y))))

;; Exemplo 14: 'do' redundante em 'loop'
(defn countdown [n]
  (loop [i n]
    (do
      (println i)
      (when (> i 0)
        (recur (dec i))))))

;; Exemplo 15: 'do' redundante em 'binding'
(defn with-dynamic-var [value]
  (binding [*print-length* 10]
    (do
      (println "Bound value:" *print-length*)
      (process-with-limit value)))) 