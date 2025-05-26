(ns test.namespaced-keys)

;; Exemplo 1: Keywords comuns que deveriam ter namespace
(def user-data
  {:id 123                    ; Deveria ser :myapp.user/id
   :name "John Doe"           ; Deveria ser :myapp.user/name
   :email "john@example.com"  ; Deveria ser :myapp.user/email
   :status "active"           ; Deveria ser :myapp.user/status
   :created-at "2023-01-01"   ; Deveria ser :myapp.user/created-at
   :role "admin"})            ; Deveria ser :myapp.user/role

;; Exemplo 2: API response sem namespace
(defn get-user-response [user]
  {:status "success"          ; Deveria ser :myapp.api/status
   :data user                 ; Deveria ser :myapp.api/data
   :message "User found"      ; Deveria ser :myapp.api/message
   :code 200})                ; Deveria ser :myapp.api/code

;; Exemplo 3: Configuração sem namespace
(def app-config
  {:database-url "jdbc:..."   ; Deveria ser :myapp.config/database-url
   :port 8080                 ; Deveria ser :myapp.config/port
   :debug true                ; Deveria ser :myapp.config/debug
   :settings {:timeout 5000}}) ; Deveria ser :myapp.config/settings

;; Exemplo 4: Spec definitions sem namespace
(require '[clojure.spec.alpha :as s])

(s/def :name string?)         ; Deveria ser :myapp.spec/name
(s/def :email string?)        ; Deveria ser :myapp.spec/email
(s/def :age pos-int?)         ; Deveria ser :myapp.spec/age

;; Exemplo 5: Database entity sem namespace
(def user-entity
  {:table :users             ; OK - contexto local
   :columns [:id :name :email :password]  ; Estas deveriam ter namespace
   :primary-key :id})         ; Deveria ser :myapp.db/id

;; Exemplo 6: Mapa grande com muitas chaves
(def complex-data
  {:id 1
   :type "document"
   :title "Important Doc"
   :content "..."
   :author "Jane"
   :tags ["important" "doc"]
   :metadata {:size 1024}
   :permissions {:read true :write false}
   :version 2
   :status "published"})

;; Exemplo 7: Keywords já namespacadas (corretas)
(def good-user-data
  {:myapp.user/id 123
   :myapp.user/name "John Doe"
   :myapp.user/email "john@example.com"
   :myapp.user/status "active"})

;; Exemplo 8: Contexto local onde namespace pode não ser necessário
(defn process-items [items]
  (let [count (count items)
        first-item (first items)
        result {:count count        ; OK - contexto local
                :first first-item   ; OK - contexto local
                :processed true}]   ; OK - contexto local
    result))

;; Exemplo 9: API routes com keywords não namespacadas
(defroute GET "/users/:id" [id]
  {:user-id id               ; Deveria ser :myapp.api/user-id
   :timestamp (System/currentTimeMillis)  ; Deveria ser :myapp.api/timestamp
   :request-id (random-uuid)}) ; Deveria ser :myapp.api/request-id

;; Exemplo 10: Error handling sem namespace
(defn handle-error [error]
  {:error true               ; Deveria ser :myapp.error/error
   :message (.getMessage error)  ; Deveria ser :myapp.error/message
   :type (type error)        ; Deveria ser :myapp.error/type
   :timestamp (System/currentTimeMillis)}) ; Deveria ser :myapp.error/timestamp 