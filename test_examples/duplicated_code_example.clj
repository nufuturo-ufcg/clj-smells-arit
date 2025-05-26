;; Exemplo de código duplicado para testar a regra duplicated-code

(ns test.duplicated-code
  (:require [clojure.string :as str]))

;; Função 1 - código que será duplicado
(defn process-user-data-v1 [user]
  (let [name (str/trim (:name user))
        email (str/lower-case (:email user))
        age (:age user)]
    (when (and (not (str/blank? name))
               (not (str/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

;; Função 2 - código muito similar (duplicado)
(defn process-customer-data [customer]
  (let [name (str/trim (:name customer))
        email (str/lower-case (:email customer))
        age (:age customer)]
    (when (and (not (str/blank? name))
               (not (str/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

;; Função 3 - mais uma duplicação
(defn validate-person-info [person]
  (let [name (str/trim (:name person))
        email (str/lower-case (:email person))
        age (:age person)]
    (when (and (not (str/blank? name))
               (not (str/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

;; Função pequena - não deve ser detectada como duplicação
(defn small-fn [x]
  (+ x 1))

;; Função refatorada - como deveria ser
(defn process-entity-data [entity]
  (let [name (str/trim (:name entity))
        email (str/lower-case (:email entity))
        age (:age entity)]
    (when (and (not (str/blank? name))
               (not (str/blank? email))
               (> age 0))
      {:processed-name name
       :processed-email email
       :processed-age age
       :status "valid"})))

;; Uso da função refatorada
(defn process-user [user]
  (process-entity-data user))

(defn process-customer [customer]
  (process-entity-data customer))

(defn validate-person [person]
  (process-entity-data person)) 