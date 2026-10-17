(ns ^{:doc "Code to use to compare collections."}
  midje.checking.checkers.collection-comparison
  (:require [clojure.math.combinatorics :as comb]
            [clojure.string :as str]
            [midje.checking.core :refer :all]
            [midje.checking.checkers
             [collection-util :refer :all]
             [util :refer :all]
             [chatty :refer :all]
             [defining :refer :all]]
            [midje.util.pile :as pile]
            [such.maps :as map]))

(defn midje-classification [x] (if (map? x) ::map ::not-map))

(defmethod collection-string ::map [_midje-classification_ elements]
  (str "{" (str/join ", " (sort elements)) "}"))

(defmethod collection-string ::not-map [_midje-classification_ elements]
  (str "[" (str/join " " elements) "]"))

(defmulti best-expected-match
  "Describe the best list of expected values found in the comparison."
  (fn [midje-classification _comparison_ _expected_] midje-classification))

(letfn [(best-expected-match-wrapper
  [midje-classification comparison expected element-maker suffix]
  (when (some inexact-checker? expected)
    [(str "      It matched: "
       (->> comparison :expected-found (map element-maker) (collection-string midje-classification))
       suffix
       ".")]))]

  (defmethod best-expected-match ::not-map [midje-classification comparison expected]
    (best-expected-match-wrapper midje-classification
      comparison
      expected
      (fn [item]
        (if (pile/named-function? item)
          (pile/function-name item)
          (pr-str item)))
      " (in that order)"))

  (defmethod best-expected-match ::map [midje-classification comparison expected]
    (best-expected-match-wrapper midje-classification
      comparison
      (vals expected)
      (fn [[k v]]
        (if (pile/named-function? v)
          (str (pr-str k) " " (pile/function-name v))
          (str (pr-str k) " " (pr-str v))))
      "")))
