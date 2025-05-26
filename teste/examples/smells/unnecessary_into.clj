(ns examples.smells.unnecessary-into
  "Exemplos de código que usa 'into' desnecessariamente quando alternativas mais idiomáticas existem")

;; Exemplo 1: Transformação de tipo desnecessária - vetor
(defn process-data [data]
  (into [] data))

;; Exemplo 2: Transformação de tipo desnecessária - conjunto
(defn unique-items [items]
  (into #{} items))

;; Exemplo 3: Transformação de tipo desnecessária - lista
(defn to-sequence [coll]
  (into () coll))

;; Exemplo 4: Mapeamento de mapa ineficiente
(defn transform-map-values [m f]
  (into {} (map (fn [[k v]] [k (f v)]) m)))

;; Exemplo 5: Mapeamento com for ineficiente
(defn process-map [data]
  (into {} (for [[k v] data] [k (* v 2)])))

;; Exemplo 6: API de transdutor não utilizada - map
(defn process-collection [coll]
  (into [] (map inc coll)))

;; Exemplo 7: API de transdutor não utilizada - filter
(defn filter-data [data]
  (into [] (filter pos? data)))

;; Exemplo 8: API de transdutor não utilizada - combinado
(defn process-numbers [numbers]
  (into #{} (map #(* % 2) numbers)))

;; Exemplo 9: Múltiplas transformações ineficientes
(defn complex-processing [data]
  (into [] (mapcat :items data)))

;; Exemplo 10: Transformação com keep
(defn extract-values [maps]
  (into [] (keep :value maps)))

;; Exemplo 11: Uso com distinct
(defn unique-values [coll]
  (into [] (distinct coll)))

;; Exemplo 12: Combinação de problemas
(defn bad-example [data]
  (into #{} (map str (filter number? data)))) 