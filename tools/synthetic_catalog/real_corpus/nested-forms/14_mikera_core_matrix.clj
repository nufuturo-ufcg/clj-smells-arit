(ns clojure.core.matrix.docgen.bench
  (:require [criterium.core :as cr]
            [hiccup.page :as h]
            [clojure.core.matrix :as m]
            [clojure.core.matrix.utils :as u]
            [clojure.core.matrix.impl.common :as c]
            [clojure.core.matrix.docgen.bench-suite :refer :all]))

(def array-types
  (array-map
   :vectorz
   {:name "vectorz"
    :constructor #(m/array :vectorz %)}
   :ndarray
   {:name "ndarray"
    :constructor #(m/array :ndarray %)}
   :ndarray-double
   {:name "ndarray-double"
    :constructor #(m/array :ndarray-double %)}
   :vecs
   {:name "persistent vectors"
    :constructor identity}))

(defn enumerated [xs]
  (map vector (range (count xs)) xs))

(defonce bench-results (atom {}))

(defn make-bench [f-name array-type to-convert bench]
  (letfn [(args-prepare [constructor args]
            (map (fn [[i arg]] (if (to-convert i) (constructor arg) arg))
                 (enumerated args)))
          (construct-bench-call [f args-name args]
            (let [arg-names (map (fn [_] (gensym)) args)
                  arg-vals (map (fn [i] `(nth ~args-name ~i))
                                (range (count args)))
                  arg-pairs (interleave arg-names arg-vals)]
              `(let [~@arg-pairs]
                 (cr/quick-benchmark (~f ~@arg-names) {}))))]
    (binding [cr/*final-gc-problem-threshold* 0.5]
      (let [f (ns-resolve 'clojure.core.matrix.protocols f-name)
            constructor (-> array-types array-type :constructor)
            arg-sets (for [[size args] bench]
                       [size ])]
        (into {} (for [[size args-raw] bench
                       :let [args (args-prepare constructor args-raw)
                             sym (->> #'dyn-args str next next (apply str) symbol)
                             bench-call (construct-bench-call f sym args)]]
                   [size (->> (binding [dyn-args args]
                                (eval bench-call))
                              :mean
                              first)]))))))

(defn perform-bench []
  (doseq [proto (u/extract-protocols)]
    (doseq [[_ {f-name :name}] (:sigs proto)
            :let [f-name-kw (keyword f-name)]]
      (doseq [[a-type a-info] array-types]
        (when-let [bench (-> @benches f-name-kw a-type)]
          (when-not (-> @bench-results f-name-kw a-type)
            (println "benchmarking" f-name "on" (name a-type))
            (swap! bench-results assoc-in [f-name-kw a-type]
                   (make-bench f-name
                               a-type
                               (-> @benches f-name-kw :to-convert)
                               (force bench))))))))
  :ok)
