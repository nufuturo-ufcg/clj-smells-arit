(ns production-doall-test)

;; Positive: the vector producer has already completed realization.
(defn redundant-mapv [items]
  (doall (mapv inc items)))

(defn redundant-filterv [items]
  (clojure.core/doall (filterv even? items)))

;; Negative: doall may be required to materialize a lazy return value.
(defn returned-map [items]
  (doall (map inc items)))

(defn returned-pmap [items]
  (doall (pmap expensive-call items)))

;; Negative: realization may be required before leaving a resource boundary.
(defn realize-before-close [reader]
  (with-open [r reader]
    (doall (map parse-line (line-seq r)))))

;; Negative: effects and synchronization make realization intentional.
(defn execute-effects [items]
  (doall (map save! items)))

(defn wait-for-workers [workers]
  (doseq [worker (doall workers)]
    @worker))

;; Negative: the producer's result contract is unknown.
(defn unknown-source [items]
  (doall (fetch-items items)))

;; Negative: local shadowing must not inherit clojure.core semantics.
(defn shadowed-doall [doall items]
  (doall (mapv inc items)))

(defn shadowed-mapv [mapv items]
  (doall (mapv inc items)))

;; Negative: partial doall is outside the exact one-argument rewrite.
(defn partial-realization [items]
  (doall 10 (mapv inc items)))

;; Negative: forms that do not execute.
(def quoted-example
  '(doall (mapv inc items)))

(comment
  (doall (mapv inc items)))

(defn comment-inside-function [items]
  (comment
    (doall (mapv inc items))))

(def syntax-template
  `(doall (mapv inc items)))
