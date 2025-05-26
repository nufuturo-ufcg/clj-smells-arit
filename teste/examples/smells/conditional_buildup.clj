(ns examples.smells.conditional-buildup
  "Exemplos de código que demonstra construção condicional excessiva")

;; Exemplo 1: Condicionais aninhadas excessivas
(defn process-grade [score]
  (if (>= score 90)
    "A"
    (if (>= score 80)
      "B"
      (if (>= score 70)
        "C"
        (if (>= score 60)
          "D"
          "F")))))

;; Exemplo 2: Cadeia de if-else que deveria usar cond
(defn get-day-type [day]
  (if (= day "monday")
    "start-of-week"
    (if (= day "friday")
      "end-of-week"
      (if (= day "saturday")
        "weekend"
        (if (= day "sunday")
          "weekend"
          "weekday")))))

;; Exemplo 3: Cadeia de igualdade que deveria usar case
(defn get-color-code [color]
  (if (= color "red")
    "#FF0000"
    (if (= color "green")
      "#00FF00"
      (if (= color "blue")
        "#0000FF"
        (if (= color "yellow")
          "#FFFF00"
          "#000000")))))

;; Exemplo 4: Aninhamento com when
(defn validate-user [user]
  (when (:active user)
    (when (:verified user)
      (when (:premium user)
        (when (> (:age user) 18)
          "valid-premium-user")))))

;; Exemplo 5: Mistura de if e when aninhados
(defn complex-validation [data]
  (if (:enabled data)
    (when (:valid data)
      (if (:processed data)
        (when (:approved data)
          "success")
        "not-processed"))
    "disabled"))

;; Exemplo 6: Condicionais profundamente aninhadas com lógica complexa
(defn process-order [order]
  (if (:customer order)
    (if (:items order)
      (if (> (count (:items order)) 0)
        (if (:payment order)
          (if (:shipping-address order)
            (if (:billing-address order)
              "order-complete"
              "missing-billing")
            "missing-shipping")
          "missing-payment")
        "no-items")
      "no-items-list")
    "no-customer"))

;; Exemplo 7: Cadeia de comparações numéricas
(defn categorize-temperature [temp]
  (if (< temp 0)
    "freezing"
    (if (< temp 10)
      "cold"
      (if (< temp 20)
        "cool"
        (if (< temp 30)
          "warm"
          "hot")))))

;; Exemplo 8: Condicionais com if-let aninhados
(defn process-nested-data [data]
  (if-let [user (:user data)]
    (if-let [profile (:profile user)]
      (if-let [settings (:settings profile)]
        (if-let [theme (:theme settings)]
          theme
          "default-theme")
        "no-settings")
      "no-profile")
    "no-user"))

;; Exemplo 9: Múltiplas condições com when-not
(defn validate-input [input]
  (when-not (nil? input)
    (when-not (empty? input)
      (when-not (< (count input) 3)
        (when-not (> (count input) 50)
          "valid-input")))))

;; Exemplo 10: Condicionais complexas com diferentes tipos
(defn determine-action [state event]
  (if (= state :idle)
    (if (= event :start)
      :running
      (if (= event :stop)
        :stopped
        :idle))
    (if (= state :running)
      (if (= event :pause)
        :paused
        (if (= event :stop)
          :stopped
          :running))
      (if (= state :paused)
        (if (= event :resume)
          :running
          (if (= event :stop)
            :stopped
            :paused))
        state)))) 