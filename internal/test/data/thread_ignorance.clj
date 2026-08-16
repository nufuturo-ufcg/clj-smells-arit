(ns thread-ignorance
  (:require [clojure.string :as string]))

(defn nested-thread-last [x]
  (vec (map inc (filter pos? (take 10 x)))))

(defn nested-thread-first [m]
  (count (keys (assoc (dissoc m :obsolete) :active true))))

(defn generic-let-thread-last [xs]
  (let [step1 (filter pos? xs)
        step2 (map inc step1)
        step3 (vec step2)]
    step3))

(defn resolved-string-alias [s]
  (count (string/split (string/trim (string/replace s #"_" " ")) #"\\s+")))
