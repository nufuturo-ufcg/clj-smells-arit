;; Exemplos de Positional Return Values
;; Este arquivo contém exemplos de funções que retornam coleções sequenciais
;; onde o significado dos elementos é implícito por sua posição.

(ns examples.smells.positional-return-values)

;; Exemplo 1: Informações de usuário como vetor
(defn get-user-info [user-id]
  ["João Silva" 30 "joao@email.com" "Desenvolvedor" "São Paulo"])

;; Exemplo 2: Coordenadas geográficas
(defn get-coordinates [address]
  [-23.5505 -46.6333]) ; latitude, longitude

;; Exemplo 3: Estatísticas de vendas
(defn calculate-sales-stats [sales-data]
  [1250.50 45 890.25 15]) ; total, count, average, returns

;; Exemplo 4: Resultado de operação com status
(defn process-payment [payment-data]
  [true "Payment processed successfully" "TXN123456" 150.00])

;; Exemplo 5: Dados de configuração
(defn get-database-config []
  ["localhost" 5432 "myapp_db" "user123" "password456"])

;; Exemplo 6: Informações de arquivo
(defn get-file-info [filepath]
  ["document.pdf" 2048576 "2023-12-01" "application/pdf"])

;; Exemplo 7: Resultado de validação
(defn validate-form [form-data]
  [false ["Email is required" "Password too short"] nil])

;; Exemplo 8: Dados de produto
(defn get-product-details [product-id]
  ["Smartphone XYZ" 899.99 "Electronics" 25 true])

;; Exemplo 9: Informações de sessão
(defn create-user-session [user]
  ["sess_abc123" 3600 "2023-12-01T10:00:00Z" false])

;; Exemplo 10: Resultado de busca
(defn search-products [query]
  [["Product A" "Product B" "Product C"] 3 0.25 true])

;; Exemplo 11: Dados de performance
(defn measure-performance [operation]
  [125.5 0.002 512 "OK"]) ; time_ms, cpu_usage, memory_mb, status

;; Exemplo 12: Informações de rede
(defn get-network-info []
  ["192.168.1.100" "255.255.255.0" "192.168.1.1" "8.8.8.8"])

;; Exemplo 13: Resultado de análise
(defn analyze-text [text]
  [150 25 8 0.85]) ; word_count, sentence_count, paragraph_count, readability

;; Exemplo 14: Dados de autenticação
(defn authenticate-user [credentials]
  [true "admin" ["read" "write" "delete"] "2023-12-01T12:00:00Z"])

;; Exemplo 15: Informações de backup
(defn create-backup [data]
  ["backup_20231201.zip" 1048576 "2023-12-01T15:30:00Z" true])

;; Exemplo 16: Retorno via let
(defn calculate-order-total [items]
  (let [subtotal (reduce + (map :price items))
        tax (* subtotal 0.1)
        shipping 10.00]
    [subtotal tax shipping (+ subtotal tax shipping)]))

;; Exemplo 17: Lista literal com múltiplos valores
(defn get-color-rgb [color-name]
  (case color-name
    "red" (255 0 0)
    "green" (0 255 0)
    "blue" (0 0 255)
    (128 128 128))) ; default gray

;; Exemplo 18: Dados de monitoramento
(defn get-system-metrics []
  [85.5 2048 1024 4 "healthy"])

;; Exemplo 19: Resultado de processamento de imagem
(defn process-image [image-data]
  [800 600 "JPEG" 245760 true])

;; Exemplo 20: Informações de transação financeira
(defn process-transaction [from-account to-account amount]
  ["TXN789012" amount "2023-12-01T16:45:00Z" "completed" 2.50]) 