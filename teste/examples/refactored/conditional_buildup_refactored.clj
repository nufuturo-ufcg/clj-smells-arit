(ns examples.refactored.conditional-buildup
  "Exemplos de código refatorado usando estruturas condicionais idiomáticas")

;; Exemplo 1: Usando cond ao invés de ifs aninhados
(defn process-grade [score]
  (cond
    (>= score 90) "A"
    (>= score 80) "B"
    (>= score 70) "C"
    (>= score 60) "D"
    :else "F"))

;; Exemplo 2: Usando cond para múltiplas condições
(defn get-day-type [day]
  (cond
    (= day "monday") "start-of-week"
    (= day "friday") "end-of-week"
    (#{day "saturday" "sunday"}) "weekend"
    :else "weekday"))

;; Exemplo 3: Usando case para cadeias de igualdade
(defn get-color-code [color]
  (case color
    "red" "#FF0000"
    "green" "#00FF00"
    "blue" "#0000FF"
    "yellow" "#FFFF00"
    "#000000"))

;; Exemplo 4: Usando and para condições sequenciais
(defn validate-user [user]
  (and (:active user)
       (:verified user)
       (:premium user)
       (> (:age user) 18)
       "valid-premium-user"))

;; Exemplo 5: Usando cond com predicados mais claros
(defn complex-validation [data]
  (cond
    (not (:enabled data)) "disabled"
    (not (:valid data)) nil
    (not (:processed data)) "not-processed"
    (:approved data) "success"
    :else nil))

;; Exemplo 6: Quebrando em funções menores
(defn has-required-fields? [order]
  (and (:customer order)
       (:items order)
       (seq (:items order))
       (:payment order)
       (:shipping-address order)
       (:billing-address order)))

(defn process-order [order]
  (cond
    (not (:customer order)) "no-customer"
    (not (seq (:items order))) "no-items"
    (not (:payment order)) "missing-payment"
    (not (:shipping-address order)) "missing-shipping"
    (not (:billing-address order)) "missing-billing"
    :else "order-complete"))

;; Exemplo 7: Usando cond para comparações numéricas
(defn categorize-temperature [temp]
  (cond
    (< temp 0) "freezing"
    (< temp 10) "cold"
    (< temp 20) "cool"
    (< temp 30) "warm"
    :else "hot"))

;; Exemplo 8: Usando some-> para navegação segura
(defn process-nested-data [data]
  (or (some-> data :user :profile :settings :theme)
      "default-theme"))

;; Exemplo 9: Usando and com predicados
(defn validate-input [input]
  (and (some? input)
       (not (empty? input))
       (>= (count input) 3)
       (<= (count input) 50)
       "valid-input"))

;; Exemplo 10: Usando mapas para state machines
(def state-transitions
  {:idle {:start :running, :stop :stopped}
   :running {:pause :paused, :stop :stopped}
   :paused {:resume :running, :stop :stopped}})

(defn determine-action [state event]
  (get-in state-transitions [state event] state))

;; Exemplo 11: Alternativa com multimethods para casos complexos
(defmulti handle-state-event (fn [state event] [state event]))

(defmethod handle-state-event [:idle :start] [_ _] :running)
(defmethod handle-state-event [:idle :stop] [_ _] :stopped)
(defmethod handle-state-event [:running :pause] [_ _] :paused)
(defmethod handle-state-event [:running :stop] [_ _] :stopped)
(defmethod handle-state-event [:paused :resume] [_ _] :running)
(defmethod handle-state-event [:paused :stop] [_ _] :stopped)
(defmethod handle-state-event :default [state _] state)

;; Exemplo 12: Usando condp para comparações com o mesmo operador
(defn categorize-score [score]
  (condp <= score
    90 "excellent"
    80 "good"
    70 "average"
    60 "below-average"
    "poor"))

;; Exemplo 13: Usando when-some para verificações de nil
(defn process-user-safely [user-data]
  (when-some [user (:user user-data)]
    (when-some [profile (:profile user)]
      (when-some [settings (:settings profile)]
        (:theme settings)))))

;; Exemplo 14: Usando if-some para casos com else
(defn get-user-theme [user-data]
  (if-some [theme (some-> user-data :user :profile :settings :theme)]
    theme
    "default-theme")) 