(ns examples.refactored.thread-ignorance
  "Exemplos de código refatorado usando threading macros para melhor legibilidade")

;; Exemplo 1: Usando -> para thread-first (o resultado vai para a primeira posição)
(defn process-user-data [user]
  (-> user
      (dissoc :password)
      (update :name str/upper-case)
      (assoc :processed true)))

;; Exemplo 2: Usando ->> para thread-last (o resultado vai para a última posição)
(defn process-collection [data]
  (->> data
       (map inc)
       (filter pos?)
       sort
       vec))

;; Exemplo 3: Threading para transformações de string
(defn clean-text [text]
  (-> text
      (str/replace #"\s+" " ")
      str/lower-case
      str/trim))

;; Exemplo 4: Threading para operações em mapas
(defn transform-config [config]
  (-> config
      (assoc :version "1.0")
      (select-keys [:name :version])
      (merge {:status :active})))

;; Exemplo 5: Usando ->> para processamento de dados
(defn analyze-data [raw-data]
  (->> raw-data
       (map #(assoc % :processed true))
       (filter :active)
       (map :value)
       (reduce +)))

;; Exemplo 6: Threading para transformações de coleção
(defn prepare-items [items]
  (->> items
       (filter :enabled)
       (sort-by :priority)
       (take 10)
       (into [])))

;; Exemplo 7: Threading para processamento de texto
(defn normalize-name [name]
  (-> name
      str/lower-case
      (str/replace #"[^a-z\s]" "")
      str/trim))

;; Exemplo 8: Threading complexo com ->>
(defn complex-transformation [data]
  (->> data
       (map #(update % :score inc))
       (filter #(> (:score %) 5))
       (mapcat :tags)
       distinct
       vec))

;; Exemplo 9: Usando some-> para threading com nil safety
(defn safe-process [user]
  (some-> user
          :profile
          :settings
          (get :theme)
          str/upper-case))

;; Exemplo 10: Usando cond-> para threading condicional
(defn conditional-process [data should-filter? should-sort?]
  (cond-> data
    should-filter? (filter :active)
    should-sort?   (sort-by :priority)
    true           vec)) 