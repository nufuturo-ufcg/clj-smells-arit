(ns examples.refactored.unnecessary-into
  "Exemplos de código refatorado usando alternativas idiomáticas ao 'into'")

;; Exemplo 1: Usando vec ao invés de (into [] ...)
(defn process-data [data]
  (vec data))

;; Exemplo 2: Usando set ao invés de (into #{} ...)
(defn unique-items [items]
  (set items))

;; Exemplo 3: Usando seq ao invés de (into () ...)
(defn to-sequence [coll]
  (seq coll))

;; Exemplo 4: Usando reduce-kv para mapeamento eficiente de mapas
(defn transform-map-values [m f]
  (reduce-kv (fn [result k v] (assoc result k (f v))) {} m))

;; Exemplo 5: Usando reduce-kv com for também
(defn process-map [data]
  (reduce-kv (fn [result k v] (assoc result k (* v 2))) {} data))

;; Exemplo 6: Usando API de transdutor com map
(defn process-collection [coll]
  (into [] (map inc) coll))

;; Exemplo 7: Usando API de transdutor com filter
(defn filter-data [data]
  (into [] (filter pos?) data))

;; Exemplo 8: Usando API de transdutor combinado
(defn process-numbers [numbers]
  (into #{} (map #(* % 2)) numbers))

;; Exemplo 9: Usando API de transdutor com mapcat
(defn complex-processing [data]
  (into [] (mapcat :items) data))

;; Exemplo 10: Usando API de transdutor com keep
(defn extract-values [maps]
  (into [] (keep :value) maps))

;; Exemplo 11: Usando API de transdutor com distinct
(defn unique-values [coll]
  (into [] (distinct) coll))

;; Exemplo 12: Usando composição de transdutores
(defn good-example [data]
  (into #{} (comp (filter number?) (map str)) data))

;; Exemplo 13: Alternativa com mapv para vetores
(defn process-to-vector [coll]
  (mapv inc coll))

;; Exemplo 14: Usando filterv para vetores
(defn filter-to-vector [data]
  (filterv pos? data))

;; Exemplo 15: Usando reduce para casos mais complexos
(defn custom-transformation [data]
  (reduce (fn [acc item]
            (if (pos? item)
              (conj acc (* item 2))
              acc))
          []
          data)) 