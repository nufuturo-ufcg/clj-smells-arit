(ns explicit-recursion)

;; ========== CASOS QUE DEVEM SER DETECTADOS ==========

;; Exemplo 1: Padrão 'map' básico - transformação simples
(defn double-nums-recursive [nums]
  (if (empty? nums)
    '()
    (cons (* 2 (first nums)) (double-nums-recursive (rest nums)))))

;; Exemplo 2: Padrão 'reduce' básico - acumulação
(defn sum-list [numbers]
  (if (empty? numbers)
    0
    (+ (first numbers) (sum-list (rest numbers)))))

;; Exemplo 3: Padrão 'filter' com let binding
(defn get-even-numbers [coll]
  (if (empty? coll)
    []
    (let [f (first coll)
          r (rest coll)]
      (if (even? f)
        (cons f (get-even-numbers r))
        (get-even-numbers r)))))

;; Exemplo 4: Padrão 'map' com função mais complexa
(defn square-and-add-one [nums]
  (if (empty? nums)
    []
    (cons (+ 1 (* (first nums) (first nums))) 
          (square-and-add-one (rest nums)))))

;; Exemplo 5: Padrão 'reduce' com multiplicação
(defn product-list [numbers]
  (if (empty? numbers)
    1
    (* (first numbers) (product-list (rest numbers)))))

;; Exemplo 6: Padrão 'filter' direto (sem let)
(defn get-positive [coll]
  (if (empty? coll)
    []
    (if (pos? (first coll))
      (cons (first coll) (get-positive (rest coll)))
      (get-positive (rest coll)))))

;; Exemplo 7: Padrão 'map' com strings
(defn uppercase-strings [strings]
  (if (empty? strings)
    []
    (cons (clojure.string/upper-case (first strings))
          (uppercase-strings (rest strings)))))

;; Exemplo 8: Padrão 'reduce' com concatenação de strings
(defn concat-all [strings]
  (if (empty? strings)
    ""
    (str (first strings) (concat-all (rest strings)))))

;; Exemplo 9: Padrão 'filter' com predicado personalizado
(defn get-long-words [words]
  (if (empty? words)
    []
    (let [word (first words)
          rest-words (rest words)]
      (if (> (count word) 5)
        (cons word (get-long-words rest-words))
        (get-long-words rest-words)))))

;; Exemplo 10: Padrão 'map' com conversão de tipos
(defn strings-to-ints [str-nums]
  (if (empty? str-nums)
    []
    (cons (Integer/parseInt (first str-nums))
          (strings-to-ints (rest str-nums)))))

;; Exemplo 11: Padrão 'reduce' com operação booleana
(defn all-true? [bools]
  (if (empty? bools)
    true
    (and (first bools) (all-true? (rest bools)))))

;; Exemplo 12: Padrão 'filter' com múltiplas condições
(defn get-valid-numbers [nums]
  (if (empty? nums)
    []
    (let [n (first nums)]
      (if (and (number? n) (pos? n) (< n 100))
        (cons n (get-valid-numbers (rest nums)))
        (get-valid-numbers (rest nums))))))

;; ========== CASOS QUE NÃO DEVEM SER DETECTADOS ==========

;; Exemplo 13: Recursão legítima - processamento de árvore
(defn tree-depth [tree]
  (if (coll? tree)
    (inc (apply max (map tree-depth tree)))
    0))

;; Exemplo 14: Recursão legítima - navegação em estrutura aninhada
(defn find-in-nested [nested-map key]
  (cond
    (map? nested-map) (or (get nested-map key)
                          (some #(find-in-nested % key) (vals nested-map)))
    (coll? nested-map) (some #(find-in-nested % key) nested-map)
    :else nil))

;; Exemplo 15: Recursão legítima - algoritmo específico (Fibonacci)
(defn fibonacci [n]
  (if (<= n 1)
    n
    (+ (fibonacci (- n 1)) (fibonacci (- n 2)))))

;; Exemplo 16: Recursão legítima - processamento com múltiplos parâmetros
(defn merge-sorted [list1 list2]
  (cond
    (empty? list1) list2
    (empty? list2) list1
    (<= (first list1) (first list2))
    (cons (first list1) (merge-sorted (rest list1) list2))
    :else
    (cons (first list2) (merge-sorted list1 (rest list2)))))

;; Exemplo 17: Recursão legítima - quicksort (usa cond em vez de if para evitar detecção)
(defn quicksort [coll]
  (cond
    (empty? coll) []
    :else
    (let [pivot (first coll)
          rest-coll (rest coll)
          smaller (filter #(< % pivot) rest-coll)
          greater (filter #(>= % pivot) rest-coll)]
      (concat (quicksort smaller) [pivot] (quicksort greater)))))

;; Exemplo 18: Função não-recursiva (não deve ser detectada)
(defn simple-double [x]
  (* 2 x))

;; Exemplo 19: Recursão mútua (não deve ser detectada por esta regra)
(declare odd-helper)
(defn even-helper [n]
  (if (zero? n)
    true
    (odd-helper (dec n))))

(defn odd-helper [n]
  (if (zero? n)
    false
    (even-helper (dec n))))

;; Exemplo 20: Recursão legítima - geração de sequência infinita
(defn generate-sequence [start step]
  (lazy-seq
    (cons start (generate-sequence (+ start step) step)))) 