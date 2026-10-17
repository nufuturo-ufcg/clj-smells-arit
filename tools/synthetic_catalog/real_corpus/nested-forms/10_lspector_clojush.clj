(ns clojush.mod_metrics
  (:use clojush.globals)
  (:require [clojure.math.numeric-tower :as math]
            [clojure.zip :as zip]
            [clojure.walk :as walk]
            [clojure.string :as string]))

(defn all-continuous-seqs
  [expr]
  (let [expr (map #(vector (first %) (second %)) expr)
        seqs (atom ())
        freq (frequencies expr)] ;; get the frequencies
    (do
      (doseq [[x y] freq]
        (doseq [k (range (dec y))] (swap! seqs conj (vector x))))

      (let [expr (distinct expr)]
        (doseq [i (range 1 (inc (count expr)))
                j (range (inc (- (count expr) i)))]
          (let [module (subvec (vec expr) j (+ j i))]
            (if (not= (reduce #(if (and (not= %1 -1) (= (first %2) (inc (last %1))))  (vector (first %1) (last %2)) -1) module) -1)
              (swap! seqs conj module)))))
      @seqs)))
