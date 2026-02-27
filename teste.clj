(ns teste
  (:require [clojure.core.async :as a]))

(def my-chan (a/chan))

(a/go
  (println "Blocked thread consuming:" (a/<!! my-chan)))


(def results (a/chan))

;; BAD SMELL: Bloquear o pool esperando por algo que depende do pool
(a/go
  (let [data (a/<!! results)] ; <!! trava a thread do pool aqui
    (println "Resultado final:" data)))

(a/go
  ;; Este bloco go pode nunca chegar a rodar se o exemplo acima 
  ;; esgotar todas as threads do pool primeiro!
  (a/>! results "Sucesso"))

(def c1 (a/chan))
(def c2 (a/chan))

;; BAD SMELL: Seleção bloqueante (alts!!) em bloco go
;; Este não foi identificado!
(a/go
  (let [[val port] (a/alts!! [c1 c2])] ; Trava a thread do pool no primeiro que responder
    (println "Recebido de" port ":" val)))

(def pipeline (a/chan 10))

;; BAD SMELL: Escrita bloqueante dentro de go
;; Este também não foi identificado!
(a/go
  (doseq [msg (range 1000)]
    ;; Quando o buffer de 10 encher, esta thread do pool para tudo
    ;; e fica esperando alguém ler do 'pipeline'.
    (a/>!! pipeline msg)
    (println "Mensagem enviada:" msg)))

(def orders (a/chan))

;; BAD SMELL: Usar <!! dentro de go
(a/go
  (while true
    (let [order (a/<!! orders)] ; <!! trava a thread do pool
      (println "Processando pedido:" order))))

;; Este não é um bad smell mas está sendo identificado como um!
(a/thread
  ;; SAFE: Aqui o bloqueio físico é permitido porque a thread 
  ;; foi criada especificamente para esta tarefa pesada.
  (println "Dedicated thread consuming:" (a/<!! my-chan)))

(a/go
  ;; GOOD: <! (um único !) "estaciona" o processo.
  ;; A thread é liberada enquanto o valor não chega.
  (let [val (a/<! my-chan)]
    (println "Non-blocking consumption:" val)))

