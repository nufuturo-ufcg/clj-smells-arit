(ns non-idiomatic-record-construction
  (:require [clojure.string :as str]))

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Interop constructor call (User.) inside defrecord definition
(defrecord User [id name email]
  Object
  (toString [_]
    (let [u (User. 1 "Ana" "ana@email.com")] ;; ERRO: Ponto após o nome dentro do defrecord
      (str u))))

;; Example 2: Constructor using "->" + name (->Order) inside defrecord definition
(defrecord Order [id total status]
  Object
  (toString [_]
    (let [o (->Order 100 50.0 :pending)] ;; ERRO: "->" + nome da função dentro do defrecord
      (str o))))

;; Example 3: Map constructor using "map->Order" inside defrecord definition
(defrecord OrderMap [id total status]
  Object
  (toString [_]
    (let [o (map->OrderMap {:id 100 :total 50.0})] ;; ERRO: "map->" + nome dentro do defrecord
      (str o))))

;; Example 4: Explicit Java 'new' constructor inside defrecord definition
(defrecord Account [id balance]
  Object
  (toString [_]
    (let [a (new Account 1 1000.0)] ;; ERRO: Instanciação interop 'new' dentro do defrecord
      (str a))))

;; Example 5: Interop constructor inside a protocol implementation within defrecord
(defprotocol Printable
  (print-info [this]))

(defrecord Person [id name]
  Printable
  (print-info [_]
    (Person. id name))) ;; ERRO: Construtor interop (Person.) dentro do defrecord

;; Example 6: Constructor using "->" + name inside a protocol implementation within defrecord
(defrecord Customer [id name]
  Printable
  (print-info [_]
    (->Customer id name))) ;; ERRO: "->" + nome dentro do protocolo no defrecord

;; Example 7: Nested interop constructor (Address.) inside defrecord definition
(defrecord Address [street city]
  Object
  (toString [_]
    (let [addr (Address. "Rua A" "SP")] ;; ERRO: Ponto no nome dentro do defrecord
      (str addr))))

;; Example 8: Interop constructor inside an anonymous fn within defrecord definition
(defrecord Task [id title]
  Object
  (toString [_]
    (let [f (fn [] (Task. id title))] ;; ERRO: Instanciação não-idiomática em função interna
      (str (f)))))

;; Example 9: Interop constructor inside a conditional inside defrecord definition
(defrecord Product [id price]
  Object
  (toString [_]
    (if (> price 0)
      (str (Product. id price)) ;; ERRO: Construtor interop na ramificação do if
      "Invalido")))

;; Example 10: Interop constructor used as value in map inside defrecord definition
(defrecord Profile [id bio]
  Object
  (toString [_]
    (let [m {:p (Profile. id bio)}] ;; ERRO: Construtor interop dentro do mapa no defrecord
      (str m))))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Idiomatic positional constructor function outside of defrecord
(defn correct-positional-constructor-example [id nome email]
  (->User id nome email))

;; Example 12: Idiomatic map-based constructor function outside of defrecord
(defn correct-map-constructor-example [dados-mapa]
  (map->Order dados-mapa))

;; Example 13: Legitimate Java native class instantiation (java.io.File) inside defrecord
(defrecord LogWriter [path]
  Object
  (toString [_]
    (.getAbsolutePath (java.io.File. path)))) ;; OK: Classe Java nativa real

;; Example 14: Legitimate Java native class instantiation (Date) inside defrecord
(defrecord Session [token]
  Object
  (toString [_]
    (str (java.util.Date.)))) ;; OK: Classe Java nativa real

;; Example 15: Legitimate Java exception instance creation inside defrecord
(defrecord Validator [input]
  Object
  (toString [_]
    (throw (Exception. "Invalido")))) ;; OK: Exceção Java nativa

;; Example 16: Java instance method invocation (.toUpperCase) inside defrecord
(defrecord Formatter [text]
  Object
  (toString [_]
    (.toUpperCase text))) ;; OK: Método Java (ponto antes do nome)

;; Example 17: Thread-first threading macro (->) inside defrecord
(defrecord Pipeline [data]
  Object
  (toString [_]
    (str (-> data
             (assoc :step 1)
             (assoc :done true))))) ;; OK: Macro de encadeamento ->

;; Example 18: Legitimate Java custom class instantiation inside defrecord
(defrecord TextBuilder [initial]
  Object
  (toString [_]
    (str (StringBuilder. initial)))) ;; OK: Classe Java nativa

;; Example 19: Custom factory function outside of defrecord
(defn custom-factory-function-example [dados]
  (assoc dados :ativo true))

;; Example 20: Safe field access on record instance using keyword
(defn record-field-access-example [instancia-usuario]
  (:name instancia-usuario))