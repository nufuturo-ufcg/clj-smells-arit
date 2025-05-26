;; Exemplos de Positional Return Values - REFATORADO
;; Este arquivo mostra como substituir retornos posicionais por mapas
;; com chaves descritivas, melhorando a legibilidade e manutenibilidade.

(ns examples.refactored.positional-return-values)

;; Exemplo 1: Informações de usuário como mapa
(defn get-user-info [user-id]
  {:name "João Silva"
   :age 30
   :email "joao@email.com"
   :role "Desenvolvedor"
   :city "São Paulo"})

;; Exemplo 2: Coordenadas geográficas
(defn get-coordinates [address]
  {:latitude -23.5505
   :longitude -46.6333})

;; Exemplo 3: Estatísticas de vendas
(defn calculate-sales-stats [sales-data]
  {:total 1250.50
   :count 45
   :average 890.25
   :returns 15})

;; Exemplo 4: Resultado de operação com status
(defn process-payment [payment-data]
  {:success true
   :message "Payment processed successfully"
   :transaction-id "TXN123456"
   :amount 150.00})

;; Exemplo 5: Dados de configuração
(defn get-database-config []
  {:host "localhost"
   :port 5432
   :database "myapp_db"
   :username "user123"
   :password "password456"})

;; Exemplo 6: Informações de arquivo
(defn get-file-info [filepath]
  {:filename "document.pdf"
   :size-bytes 2048576
   :created-date "2023-12-01"
   :mime-type "application/pdf"})

;; Exemplo 7: Resultado de validação
(defn validate-form [form-data]
  {:valid false
   :errors ["Email is required" "Password too short"]
   :data nil})

;; Exemplo 8: Dados de produto
(defn get-product-details [product-id]
  {:name "Smartphone XYZ"
   :price 899.99
   :category "Electronics"
   :stock 25
   :available true})

;; Exemplo 9: Informações de sessão
(defn create-user-session [user]
  {:session-id "sess_abc123"
   :expires-in 3600
   :created-at "2023-12-01T10:00:00Z"
   :remember-me false})

;; Exemplo 10: Resultado de busca
(defn search-products [query]
  {:results ["Product A" "Product B" "Product C"]
   :total-count 3
   :search-time 0.25
   :has-more false})

;; Exemplo 11: Dados de performance
(defn measure-performance [operation]
  {:time-ms 125.5
   :cpu-usage 0.002
   :memory-mb 512
   :status "OK"})

;; Exemplo 12: Informações de rede
(defn get-network-info []
  {:ip-address "192.168.1.100"
   :subnet-mask "255.255.255.0"
   :gateway "192.168.1.1"
   :dns "8.8.8.8"})

;; Exemplo 13: Resultado de análise
(defn analyze-text [text]
  {:word-count 150
   :sentence-count 25
   :paragraph-count 8
   :readability-score 0.85})

;; Exemplo 14: Dados de autenticação
(defn authenticate-user [credentials]
  {:authenticated true
   :username "admin"
   :permissions ["read" "write" "delete"]
   :login-time "2023-12-01T12:00:00Z"})

;; Exemplo 15: Informações de backup
(defn create-backup [data]
  {:filename "backup_20231201.zip"
   :size-bytes 1048576
   :created-at "2023-12-01T15:30:00Z"
   :success true})

;; Exemplo 16: Retorno via let com mapa
(defn calculate-order-total [items]
  (let [subtotal (reduce + (map :price items))
        tax (* subtotal 0.1)
        shipping 10.00
        total (+ subtotal tax shipping)]
    {:subtotal subtotal
     :tax tax
     :shipping shipping
     :total total}))

;; Exemplo 17: Cores RGB como mapa
(defn get-color-rgb [color-name]
  (case color-name
    "red" {:red 255 :green 0 :blue 0}
    "green" {:red 0 :green 255 :blue 0}
    "blue" {:red 0 :green 0 :blue 255}
    {:red 128 :green 128 :blue 128})) ; default gray

;; Exemplo 18: Dados de monitoramento
(defn get-system-metrics []
  {:cpu-usage-percent 85.5
   :memory-total-mb 2048
   :memory-used-mb 1024
   :active-processes 4
   :status "healthy"})

;; Exemplo 19: Resultado de processamento de imagem
(defn process-image [image-data]
  {:width 800
   :height 600
   :format "JPEG"
   :size-bytes 245760
   :processed true})

;; Exemplo 20: Informações de transação financeira
(defn process-transaction [from-account to-account amount]
  {:transaction-id "TXN789012"
   :amount amount
   :timestamp "2023-12-01T16:45:00Z"
   :status "completed"
   :fee 2.50}) 