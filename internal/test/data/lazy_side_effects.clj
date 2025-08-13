(ns lazy-side-effects)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: 'map' with printing (side effect inside lazy sequence)
(defn notify-map [nums]
  (map #(do (println "Notifying:" %) %) nums))

;; Example 2: 'filter' with printing
(defn filter-even-with-print [nums]
  (filter #(do (println "Checking:" %) (even? %)) nums))

;; Example 3: Lazy 'for' comprehension with printing
(defn lazy-for-print [nums]
  (for [n nums] (do (println "For loop value:" n) n)))

;; Example 4: 'map' with atom swap! (state mutation) (not detected)
(def counter (atom 0))
(defn increment-atom-lazy [nums]
  (map #(swap! counter inc) nums))

;; Example 5: 'filter' with swap! (state mutation) (not detected)
(defn filter-and-increment [nums]
  (filter #(do (swap! counter inc) (even? %)) nums))


;; Example 7: 'take' on lazy sequence with side effect
(defn take-lazy-with-print [nums]
  (take 2 (map #(do (println "Taking:" %) %) nums)))

;; Example 8: 'drop' on lazy sequence with side effect
(defn drop-lazy-with-print [nums]
  (drop 2 (map #(do (println "Dropping:" %) %) nums)))

;; Example 9: 'map' sending message to agent (side effect) (not detected)
(def a (agent 0))
(defn send-agent-lazy [nums]
  (map #(send a + %) nums))

;; Example 10: 'map' writing to file (side effect) (not detected)
(defn write-file-lazy [nums]
  (map #(spit "output.txt" (str % "\n") :append true) nums))

;; Example 11: 'for' with swap! (state mutation in lazy comprehension)
(defn lazy-for-swap [nums]
  (for [n nums] (swap! counter + n)))

;; Example 12: 'map' with side effect and partial realization
(defn partial-realization [nums]
  (first (map #(do (println "Partial:" %) %) nums)))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 13: Printing outside lazy sequence
(defn eager-print [nums]
  (doseq [n nums] (println "Eager:" n))
  nums)

;; Example 14: swap! outside lazy evaluation
(defn eager-swap [nums]
  (doseq [n nums] (swap! counter + n))
  nums)

;; Example 15: Fully realized 'map' with side effect (realized immediately)
(defn realized-map-print [nums]
  (doall (map #(println "Realized:" %) nums)))

;; Example 16: Lazy sequence without side effects
(defn lazy-no-side-effect [nums]
  (map #(* 2 %) nums))

;; Example 17: Recursive function with side effect (non-lazy)
(defn recursive-print [nums]
  (if (empty? nums)
    '()
    (do (println "Recursive:" (first nums))
        (cons (first nums) (recursive-print (rest nums))))))

;; Example 18: Using 'for' comprehension without side effects
(defn lazy-for-no-side-effect [nums]
  (for [n nums] (* 2 n)))

;; Example 19: Partial lazy sequence realized fully (no smell)
(defn full-realization [nums]
  (doall (map #(swap! counter inc) nums)))

;; Example 20: Side effect inside let but outside lazy
(defn side-effect-outside-lazy [nums]
  (let [n (count nums)]
    (println "Count is:" n)
    nums))

