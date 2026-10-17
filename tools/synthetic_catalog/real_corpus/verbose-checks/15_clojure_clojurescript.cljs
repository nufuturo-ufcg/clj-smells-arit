(ns cljs.pprint
  (:refer-clojure :exclude [deftype print println pr prn float?])
  (:require-macros
    [cljs.pprint :as m :refer [with-pretty-writer getf setf deftype
                               pprint-logical-block print-length-loop
                               defdirectives formatter-out]])
  (:require
    [cljs.core :refer [IWriter IDeref]]
    [clojure.string :as string]
    [goog.string :as gstring])
  (:import [goog.string StringBuffer]))

(defn char-code
  "Convert char to int"
  [c]
  (cond
    (number? c) c
    (and (string? c) (== (.-length c) 1)) (.charCodeAt c 0)
    :else (throw (js/Error. "Argument to char must be a character or number"))))

(defn- inc-s
  "Assumption: The input string consists of one or more decimal digits,
  and no other characters. Return a string containing one or more
  decimal digits containing a decimal number one larger than the input
  string. The output string will always be the same length as the input
  string, or one character longer."
  [s]
  (let [len-1 (dec (count s))]
    (loop [i (int len-1)]
      (cond
        (neg? i) (apply str "1" (repeat (inc len-1) "0"))
        (= \9 (.charAt s i)) (recur (dec i))
        :else (apply str (subs s 0 i)
                     (char (inc (char-code (.charAt s i))))
                     (repeat (- len-1 i) "0"))))))

(defn- round-str [m e d w]
  (if (or d w)
    (let [len (count m)
          ;; Every formatted floating point number should include at
          ;; least one decimal digit and a decimal point.
          w (if w (max 2 w)
                  ;;NB: if w doesn't exist, it won't ever be used because d will
                  ;; satisfy the cond below. cljs gives a compilation warning if
                  ;; we don't provide a value here.
                  0)
          round-pos (cond
                      ;; If d was given, that forces the rounding
                      ;; position, regardless of any width that may
                      ;; have been specified.
                      d (+ e d 1)
                      ;; Otherwise w was specified, so pick round-pos
                      ;; based upon that.
                      ;; If e>=0, then abs value of number is >= 1.0,
                      ;; and e+1 is number of decimal digits before the
                      ;; decimal point when the number is written
                      ;; without scientific notation. Never round the
                      ;; number before the decimal point.
                      (>= e 0) (max (inc e) (dec w))
                      ;; e < 0, so number abs value < 1.0
                      :else (+ w e))
          [m1 e1 round-pos len] (if (= round-pos 0)
                                  [(str "0" m) (inc e) 1 (inc len)]
                                  [m e round-pos len])]
      (if round-pos
        (if (neg? round-pos)
          ["0" 0 false]
          (if (> len round-pos)
            (let [round-char (nth m1 round-pos)
                  result (subs m1 0 round-pos)]
              (if (>= (char-code round-char) (char-code \5))
                (let [round-up-result (inc-s result)
                      expanded (> (count round-up-result) (count result))]
                  [(if expanded
                     (subs round-up-result 0 (dec (count round-up-result)))
                     round-up-result)
                   e1 expanded])
                [result e1 false]))
            [m e false]))
        [m e false]))
    [m e false]))
