(ns test.explicit-recursion)

;; Exemplo 1: Padrão de acumulação - deveria usar reduce
(defn sum-numbers [numbers]
  (loop [acc 0
         coll numbers]
    (if (empty? coll)
      acc
      (recur (+ acc (first coll))
             (rest coll)))))

;; Exemplo 2: Padrão de transformação - deveria usar map
(defn transform-items [items]
  (loop [result []
         coll items]
    (if (empty? coll)
      result
      (recur (conj result (str (first coll)))
             (rest coll)))))

;; Exemplo 3: Padrão de filtro - deveria usar filter
(defn filter-positive [numbers]
  (loop [result []
         coll numbers]
    (if (empty? coll)
      result
      (if (pos? (first coll))
        (recur (conj result (first coll))
               (rest coll))
        (recur result
               (rest coll))))))

;; Exemplo 4: Padrão de contagem - deveria usar count
(defn count-items [items]
  (loop [cnt 0
         coll items]
    (if (empty? coll)
      cnt
      (recur (inc cnt)
             (rest coll)))))

;; Exemplo 5: Iteração simples - deveria usar doseq
(defn print-all [items]
  (loop [coll items]
    (when (not (empty? coll))
      (println (first coll))
      (recur (rest coll)))))

;; Exemplo 6: Recursão complexa que pode ser apropriada
(defn complex-calculation [data depth]
  (loop [current data
         level depth
         cache {}]
    (if (zero? level)
      current
      (let [processed (process-complex current cache)]
        (if (should-continue? processed)
          (recur processed
                 (dec level)
                 (update-cache cache processed))
          processed)))))

;; Exemplo 7: Acumulação com merge - deveria usar reduce
(defn merge-maps [maps]
  (loop [result {}
         coll maps]
    (if (empty? coll)
      result
      (recur (merge result (first coll))
             (rest coll)))))

;; Exemplo 8: Transformação com condição - pode usar map + filter
(defn transform-and-filter [items]
  (loop [result []
         coll items]
    (if (empty? coll)
      result
      (let [item (first coll)]
        (if (valid? item)
          (recur (conj result (transform item))
                 (rest coll))
          (recur result
                 (rest coll))))))) 