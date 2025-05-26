(ns test.namespaced-keys-portability
  "Exemplos demonstrando questões de portabilidade de keywords")

;; Exemplo 1: Keywords que serão usadas em APIs REST/GraphQL
;; Estas precisam ser portáveis para JSON, então snake_case seria melhor
(def api-user-data
  {:user-id 123              ; Problemático: user-id não é válido em muitos contextos
   :first-name "John"        ; Problemático: first-name pode causar problemas
   :last-name "Doe"          ; Problemático: last-name pode causar problemas
   :email-address "john@example.com"  ; Problemático para APIs
   :phone-number "+1234567890"})      ; Problemático para APIs

;; Melhor seria:
(def portable-user-data
  {:myapp_user/user_id 123
   :myapp_user/first_name "John"
   :myapp_user/last_name "Doe"
   :myapp_user/email_address "john@example.com"
   :myapp_user/phone_number "+1234567890"})

;; Exemplo 2: Keywords para banco de dados
;; Estas se tornarão colunas SQL, então precisam ser compatíveis
(def database-schema
  {:table-name "users"       ; Problemático para SQL
   :primary-key :user-id     ; Problemático para SQL
   :foreign-keys [:company-id :department-id]  ; Problemático para SQL
   :indexes [:email-address :created-at]})     ; Problemático para SQL

;; Melhor seria:
(def portable-schema
  {:myapp_db/table_name "users"
   :myapp_db/primary_key :myapp_user/user_id
   :myapp_db/foreign_keys [:myapp_company/company_id :myapp_dept/department_id]
   :myapp_db/indexes [:myapp_user/email_address :myapp_user/created_at]})

;; Exemplo 3: Configuração que será exportada para diferentes formatos
(def app-configuration
  {:database-url "jdbc:postgresql://..."  ; Problemático para env vars
   :redis-host "localhost"                ; Problemático para env vars
   :max-connections 10                    ; Problemático para env vars
   :connection-timeout 5000               ; Problemático para env vars
   :retry-attempts 3})                    ; Problemático para env vars

;; Melhor seria:
(def portable-configuration
  {:myapp_config/database_url "jdbc:postgresql://..."
   :myapp_config/redis_host "localhost"
   :myapp_config/max_connections 10
   :myapp_config/connection_timeout 5000
   :myapp_config/retry_attempts 3})

;; Exemplo 4: Dados que serão serializados para ElasticSearch
(def search-document
  {:document-id "doc123"     ; Problemático para ES
   :content-type "text/html" ; Problemático para ES
   :created-by "user456"     ; Problemático para ES
   :last-modified "2023-01-01"  ; Problemático para ES
   :search-tags ["important" "document"]})  ; Problemático para ES

;; Melhor seria:
(def portable-document
  {:myapp_search/document_id "doc123"
   :myapp_search/content_type "text/html"
   :myapp_search/created_by "user456"
   :myapp_search/last_modified "2023-01-01"
   :myapp_search/search_tags ["important" "document"]})

;; Exemplo 5: Keywords que permanecerão apenas em Clojure (OK usar lisp-case)
(defn process-data [data]
  (let [temp-result (transform data)
        final-result (validate temp-result)]
    {:processing-time (System/currentTimeMillis)  ; OK - contexto local
     :validation-passed? (valid? final-result)    ; OK - contexto local
     :temp-data temp-result                       ; OK - contexto local
     :final-data final-result}))                  ; OK - contexto local

;; Exemplo 6: Spec definitions que podem ser compartilhadas
(require '[clojure.spec.alpha :as s])

;; Problemático - specs podem ser usadas em outras linguagens via babashka/sci
(s/def :user-id pos-int?)
(s/def :first-name string?)
(s/def :email-address (s/and string? #(re-matches #".+@.+" %)))

;; Melhor seria:
(s/def :myapp_spec/user_id pos-int?)
(s/def :myapp_spec/first_name string?)
(s/def :myapp_spec/email_address (s/and string? #(re-matches #".+@.+" %)))

;; Exemplo 7: GraphQL schema mapping
(def graphql-user-type
  {:type-name "User"
   :fields {:user-id {:type "ID!"}          ; Problemático para GraphQL
            :first-name {:type "String!"}   ; Problemático para GraphQL
            :email-address {:type "String!"} ; Problemático para GraphQL
            :created-at {:type "DateTime!"}}}) ; Problemático para GraphQL 