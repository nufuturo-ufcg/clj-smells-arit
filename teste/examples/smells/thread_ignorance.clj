(ns examples.smells.thread-ignorance
  "Exemplos de código que ignora threading macros e seria mais legível com -> ou ->>")

;; Exemplo 1: Aninhamento que se beneficiaria de ->
(defn process-user-data [user]
  (assoc (update (dissoc user :password) :name str/upper-case) :processed true))

;; Exemplo 2: Múltiplas transformações aninhadas que se beneficiariam de ->>
(defn process-collection [data]
  (vec (sort (filter pos? (map inc data)))))

;; Exemplo 3: Transformações de string aninhadas
(defn clean-text [text]
  (str/trim (str/lower-case (str/replace text #"\s+" " "))))

;; Exemplo 4: Operações em mapas aninhadas
(defn transform-config [config]
  (merge (select-keys (assoc config :version "1.0") [:name :version]) {:status :active}))

;; Exemplo 5: Processamento de dados complexo
(defn analyze-data [raw-data]
  (reduce + (map :value (filter :active (map #(assoc % :processed true) raw-data)))))

;; Exemplo 6: Transformações de coleção com múltiplos passos
(defn prepare-items [items]
  (into [] (take 10 (sort-by :priority (filter :enabled items)))))

;; Exemplo 7: Processamento de texto com múltiplas etapas
(defn normalize-name [name]
  (str/trim (str/replace (str/lower-case name) #"[^a-z\s]" "")))

;; Exemplo 8: Aninhamento profundo com diferentes tipos de operações
(defn complex-transformation [data]
  (vec (distinct (mapcat :tags (filter #(> (:score %) 5) (map #(update % :score inc) data)))))) 