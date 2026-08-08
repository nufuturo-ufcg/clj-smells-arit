(ns blocking-inside-go
  (:require [clojure.core.async :as a]))

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Basic channel read blocking (<!!) inside go
(defn read-blocking-example [chan1]
  (a/go
    (let [dado (a/<!! chan1)]
      (println dado))))

;; Example 2: Basic channel write blocking (>!!) inside go
(defn write-blocking-example [chan-out]
  (a/go
    (a/>!! chan-out :sucesso)))

;; Example 3: Java thread sleep blocking inside go
(defn sleep-blocking-example []
  (a/go
    (Thread/sleep 1000)
    (println "Acordou!")))

;; Example 4: Choice selection blocking (alts!!) inside go
(defn alts-blocking-example [chan1 chan2]
  (a/go
    (let [[v ch] (a/alts!! [chan1 chan2])]
      (println "Recebido de:" ch))))

;; Example 5: Alternative choice macro blocking (alt!!) inside go
(defn alt-macro-blocking-example [ch1 ch2]
  (a/go
    (a/alt!!
      ch1 ([v] (println "Lido:" v))
      ch2 ([v] (println "Lido:" v)))))

;; Example 6: Synchronous database/HTTP emulation using sleep inside go
(defn db-io-blocking-example [canal-saida]
  (a/go
    (Thread/sleep 2000)
    (a/>! canal-saida :dados)))

;; Example 7: Loop structure containing thread sleep inside go
(defn loop-sleep-blocking-example []
  (a/go
    (doseq [x [1 2 3]]
      (Thread/sleep 500)
      (println x))))

;; Example 8: Hidden blocking operation via indirect function call (False Negative)
(defn buscar-dados-legado []
  (Thread/sleep 5000)
  :dados)

(defn hidden-function-blocking-example [canal-saida]
  (a/go
    (let [resultado (buscar-dados-legado)]
      (a/>! canal-saida resultado))))

;; Example 9: Direct blocking write (>!!) inside an if condition
(defn conditional-blocking-example [ch msg condicao]
  (a/go
    (if condicao
      (a/>!! ch msg) ;; ERRO: Operação de bloqueio ativa dentro do fluxo do go
      (println "Ignorado"))))

;; Example 10: Nested expression execution with blocking operation inside let
(defn nested-expr-blocking-example [ch]
  (a/go
    (let [val (identity :teste)
          dado (a/<!! ch)] ;; ERRO: O linter deve detectar a var síncrona dentro do escopo let
      (println val dado))))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Correct parking operation using (<!) inside go
(defn correct-parking-read-example [ch]
  (a/go
    (let [msg (a/<! ch)]
      (println msg))))

;; Example 12: Correct parking operation using (>!) inside go
(defn correct-parking-write-example [ch dado]
  (a/go
    (a/>! ch dado)))

;; Example 13: Legitimate blocking operation inside a dedicated 'a/thread' (False Positive)
(defn thread-macro-safe-example [canal-longo]
  (a/go
    (println "Iniciando processo no go block...")
    (a/thread
      (let [resultado (a/<!! canal-longo)]
        (println "Processado fora do pool go:" resultado)))))

;; Example 14: Legitimate blocking operation inside a 'future' (False Positive)
(defn future-macro-safe-example [ch]
  (a/go
    (future
      (a/<!! ch))))

;; Example 15: Un-evaluated 'delay' enclosing blocking action (False Positive)
(defn delay-safe-example [ch]
  (a/go
    (let [d (delay (a/<!! ch))])
    (println "Delay definido")))

;; Example 16: Un-realized lazy sequence wrapping blocking operation (False Positive)
(defn lazy-seq-safe-example [ch]
  (a/go
    (let [s (lazy-seq (cons (a/<!! ch) nil))])))

;; Example 17: Anonymous function definition containing blocking action (False Positive)
(defn anonymous-fn-safe-example [ch]
  (a/go
    (let [minha-funcao (fn [] (a/<!! ch))]
      (println "Função criada, mas não invocada!"))))

;; Example 18: Local function definition 'letfn' enclosing blocking call (False Positive)
(defn letfn-safe-example [ch]
  (a/go
    (letfn [(funcao-local [] (a/<!! ch))])
    (println "Declarada com letfn")))

;; Example 19: Pure non-blocking operations outside of a go block context
(defn simple-math-operation [x]
  (* x x))

;; Example 20: Safe parking loop over multiple channels inside go
(defn multi-channel-parking-example [chan1 chan2 chan3]
  (a/go
    (doseq [ch [chan1 chan2 chan3]]
      (let [msg (a/<! ch)]
        (println "Processado:" msg)))))