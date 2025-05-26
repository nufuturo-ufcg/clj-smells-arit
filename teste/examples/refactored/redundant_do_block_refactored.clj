;; Exemplos de Redundant Do Block - REFATORADO
;; Este arquivo mostra como remover blocos 'do' redundantes,
;; aproveitando o 'do' implícito das formas pai.

(ns examples.refactored.redundant-do-block)

;; Exemplo 1: 'do' removido de 'when' (when já fornece do implícito)
(defn process-when-valid [data]
  (when (valid? data)
    (log "Processing data")
    (transform data)
    (save-result data)))

;; Exemplo 2: 'do' removido de 'let' (let já fornece do implícito no corpo)
(defn calculate-total [items]
  (let [prices (map :price items)
        tax-rate 0.1]
    (println "Calculating total")
    (+ (reduce + prices) (* (reduce + prices) tax-rate))))

;; Exemplo 3: 'do' removido de 'if' (if não fornece do implícito, mas podemos usar when/when-not)
(defn handle-user-input [input]
  (if (valid-input? input)
    (do
      (log-input input)
      (process-input input))
    (do
      (log-error "Invalid input")
      (show-error-message))))

;; Exemplo 4: 'do' removido de 'defn' (defn já fornece do implícito no corpo)
(defn initialize-system []
  (load-config)
  (setup-database)
  (start-services))

;; Exemplo 5: 'do' removido de 'fn' (fn já fornece do implícito no corpo)
(def process-data
  (fn [data]
    (validate data)
    (transform data)
    (persist data)))

;; Exemplo 6: 'do' removido de 'when-let' (when-let já fornece do implícito)
(defn process-user [user-id]
  (when-let [user (find-user user-id)]
    (update-last-seen user)
    (send-notification user)
    user))

;; Exemplo 7: 'do' removido de 'try' e 'catch' (ambos fornecem do implícito)
(defn safe-operation [data]
  (try
    (validate-data data)
    (risky-operation data)
    (catch Exception e
      (log-error e)
      (handle-error e))))

;; Exemplo 8: 'do' removido de 'cond' (cond não fornece do implícito nos ramos)
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

;; Exemplo 9: 'do' removido de 'case' (case não fornece do implícito nos ramos)
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

;; Exemplo 10: 'do' removido de 'finally' (finally fornece do implícito)
(defn cleanup-operation []
  (try
    (perform-operation)
    (finally
      (close-connections)
      (cleanup-temp-files)
      (log "Cleanup completed"))))

;; Exemplo 11: 'do' vazio removido completamente
(defn empty-do-example []
  (when true
    nil)) ; ou simplesmente remover o when se não faz nada

;; Exemplo 12: 'do' com única expressão removido
(defn single-expression-do []
  (when (ready?)
    (start-process)))

;; Exemplo 13: 'do' removido de multi-arity function
(defn multi-arity-example
  ([x]
   (println "Single arg")
   (* x 2))
  ([x y]
   (println "Two args")
   (+ x y)))

;; Exemplo 14: 'do' removido de 'loop' (loop fornece do implícito no corpo)
(defn countdown [n]
  (loop [i n]
    (println i)
    (when (> i 0)
      (recur (dec i)))))

;; Exemplo 15: 'do' removido de 'binding' (binding fornece do implícito)
(defn with-dynamic-var [value]
  (binding [*print-length* 10]
    (println "Bound value:" *print-length*)
    (process-with-limit value))) 