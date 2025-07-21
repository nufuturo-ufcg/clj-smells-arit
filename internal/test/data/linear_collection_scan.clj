#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns lcs)

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;; LINEAR COLLECTION SCAN - EXEMPLARES CLÁSSICOS
;; Exemplos clássicos de varredura de coleções em Clojure
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;

;;--------------------------------------------------
;; Exemplo 1: Encontrar elemento usando loop manual (ruim)
(defn scan-find-bad [pred coll]
  (loop [c coll]
    (when (seq c)
      (if (pred (first c))
        (first c)
        (recur (rest c))))))

;; Refatorado: Usando some (idiomático)
(defn scan-find-good [pred coll]
  (some #(when (pred %) %) coll))

;;--------------------------------------------------
;; Exemplo 2: Contar elementos que satisfazem predicado (ruim)
(defn scan-count-bad [coll pred]
  (count (filter pred coll)))

;; Refatorado: Usando transduce (eficiente)
(defn scan-count-good [coll pred]
  (transduce (filter pred) (completing (fn [acc _] (inc acc))) 0 coll))

;;--------------------------------------------------
;; Exemplo 3: Encontrar mínimo usando sort (ruim)
(defn scan-min-bad [coll]
  (first (sort coll)))

;; Refatorado: Usando apply/min (eficiente)
(defn scan-min-good [coll]
  (apply min coll))

;;--------------------------------------------------
;; Exemplo 4: Checar existência usando filter+count (ruim)
(defn scan-exists-bad [x coll]
  (> (count (filter #(= % x) coll)) 0))

;; Refatorado: Usando some (eficiente)
(defn scan-exists-good [x coll]
  (some #(= % x) coll))

;;--------------------------------------------------
;; Exemplo 5: Múltiplos maps encadeados (ruim)
(defn scan-multi-map-bad [coll]
  (map inc (map #(* % 2) (map abs coll))))

;; Refatorado: Composição de funções (eficiente)
(defn scan-multi-map-good [coll]
  (map (comp inc #(* % 2) abs) coll))

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;; END OF FILE
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;