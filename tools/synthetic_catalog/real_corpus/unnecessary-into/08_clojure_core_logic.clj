(ns clojure.core.logic.dcg
  (:refer-clojure :exclude [==])
  (:use [clojure.core.logic]))

(defn lsym [n]
  (gensym (str "l" n "_")))

(defn !dcg? [clause]
  (and (sequential? clause)
       (let [f (first clause)]
         (and (symbol? f)
              (= (name f) "!dcg")))))

(defn fresh-expr? [clause]
  (and (seq? clause)
       (let [f (first clause)]
         (and (symbol? f)
              (= (name f) "fresh")))))

(defn count-clauses [clauses]
  (if (fresh-expr? clauses)
    (count-clauses (drop 2 clauses))
    (reduce (fn [s c]
              (cond
               (fresh-expr? c) (+ s (count-clauses (drop 2 c)))
               (!dcg? c) s
               :else (clojure.core/inc s)))
            0 clauses)))

(defn mark-clauses
  ([cs] (mark-clauses cs (atom 0)))
  ([[c & r :as cs] i]
     (cond
      (nil? (seq cs)) ()
      (fresh-expr? c) (cons `(fresh ~(second c)
                          ~@(mark-clauses (drop 2 c) i))
                       (mark-clauses r i))
      (!dcg? c) (cons c (mark-clauses r i))
      :else (cons (with-meta c
                    {:index (swap! i clojure.core/inc)})
                  (mark-clauses r i)))))

(defn handle-clauses [env [c & r :as cs]]
  (cond
   (nil? (seq cs)) ()
   (fresh-expr? c) (cons `(fresh ~(second c)
                       ~@(handle-clauses env (drop 2 c)))
                    (handle-clauses env r))
   (!dcg? c) (cons (second c) (handle-clauses env r))
   (vector? c) (cons (->lcons env c (-> c meta :index))
                     (handle-clauses env r))
   (and (seq? c)
        (= (first c) `quote)
        (vector? (second c))) (cons (->lcons env (second c) (-> c meta :index) true)
                                     (handle-clauses env r))
   :else (let [i (-> c meta :index)
               c (if (seq? c) c (list c))]
           (cons (concat c [(env (dec i)) (env i)])
                 (handle-clauses env r)))))

(defmacro --> [name & clauses]
  (let [r (range 1 (+ (count-clauses clauses) 2))
        lsyms (into [] (map lsym r))
        clauses (mark-clauses clauses)
        clauses (handle-clauses lsyms clauses)]
    `(defn ~name [~(first lsyms) ~(last lsyms)]
       (fresh [~@(butlast (rest lsyms))]
         ~@clauses))))
