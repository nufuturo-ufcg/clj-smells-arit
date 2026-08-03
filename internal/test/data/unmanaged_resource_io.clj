(ns unmanaged-resource-io)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: single resource instantiation without with-open
(defn ler-config [caminho]
  (let [r (clojure.java.io/reader caminho)]
    (let [linhas (line-seq r)]
      (doall linhas))))

;; Example 2: output stream opened and left unmanaged
(defn salvar-bytes [caminho dados]
  (let [out (clojure.java.io/output-stream caminho)]
    (.write out dados)))

;; Example 3: direct Java interop instantiation of a Closeable class
(defn processa-raw [path]
  (let [f-reader (java.io.FileReader. path)]
    (.read f-reader)))

;; Example 4: server socket instantiation without resource management
(defn iniciar-escuta [porta]
  (let [srv (java.net.ServerSocket. porta)]
    (.accept srv)))

;; Example 5: stream leakage inside an anonymous function but executed immediately
(defn processar-imediato [caminho]
  ((fn []
     (let [stream (clojure.java.io/input-stream caminho)]
       (.read stream)))))

;; Example 6: custom alias used for clojure.java.io that still leaks a reader
(defn leak-with-alias [caminho]
  (let [f (clojure.java.io/reader caminho)]
    #_{:clj-kondo/ignore [:invalid-arity]}
    (read-line f)))

;; Example 7: zip compression stream instantiated loosely
(defn descompactar [arquivo-zip]
  (let [zip (java.util.zip.GZIPInputStream. (clojure.java.io/input-stream arquivo-zip))]
    (.read zip)))

;; Example 8: socket connection created without explicit cleanup block
(defn vazamento-rede []
  (let [canal (java.net.Socket. "127.0.0.1" 9000)]
    (.isConnected canal)))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 1: correct usage of with-open macro for a single reader
(defn ler-config-seguro [caminho]
  (with-open [r (clojure.java.io/reader caminho)]
    (count (line-seq r))))

;; Example 2: correct usage of with-open binding multiple resources
(defn copiar-arquivo [origem destino]
  (with-open [in (clojure.java.io/input-stream origem)
              out (clojure.java.io/output-stream destino)]
    (clojure.java.io/copy in out)))

;; Example 3: standard console stream output that should never be closed
(defn escrever-no-console [dados]
  (let [w (clojure.java.io/writer System/out)]
    (.write w (str dados))))

;; Example 4: safe manual resource management using equivalent try/finally
(defn gerencia-manual [caminho]
  (let [stream (clojure.java.io/input-stream caminho)]
    (try
      (.read stream)
      (finally
        (.close stream)))))

;; Example 5: resource passed as argument, lifecycle belongs to the caller
(defn processar-dados-correntes [^java.io.Reader input-reader]
  (.read input-reader))

;; Example 6: data structure in memory implementing Closeable but holding no OS file descriptors
(defn obter-string-reader [texto]
  (java.io.StringReader. texto))

;; Example 7: factory function intended to return the resource for external management
(defn criar-conexao-temporaria [url]
  (java.net.Socket. url 80))

;; Example 8: regular let block handling non-I/O Clojure data structures
(defn simple-let []
  (let [a 1
        b 2]
    (+ a b)))