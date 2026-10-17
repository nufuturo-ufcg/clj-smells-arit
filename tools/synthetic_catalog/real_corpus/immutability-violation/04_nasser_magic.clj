(ns clojure.teststm)

(defn sleep [time-ms] (System.Threading.Thread/Sleep time-ms))

(def a (ref #{}))

(def b (ref #{}))

(def cycle-continue true)

(defn make-future [id]
  (future
   (try
     (loop [n 0]
       ;; join the party
       (dosync
        (alter a conj id)
        (alter b conj id))

       ;; validate the refs
       (dosync
        (let [current-a @a
              current-b @b]
          (if (not (= current-a current-b))
            (throw (Exception. (str (format "\n%s\n%s" current-a current-b)))))))

       ;; leave
       (dosync
        (alter a disj id)
        (alter b disj id))

       (if cycle-continue
         (recur (inc n))))
     (catch Exception ex
       (def cycle-continue false)
       (sleep 100)
       (println ex)))))

(defn f1 [nagts dur]
  (future
    (do
      (def a (ref #{}))
      (def b (ref #{}))
      (def cycle-continue true)
      (let [n-agents nagts
            duration dur
            futures (doall (map make-future (range n-agents)))]
        (loop [i 0]
          (sleep 1000)
          (print ".") (flush)
          (if (and (<= i duration) cycle-continue)
            (recur (inc i)))))
      (println "done")
      (def cycle-continue false))))
