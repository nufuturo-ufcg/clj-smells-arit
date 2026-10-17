(ns clojure.core.matrix.macros-clj
  "Namespace for core.matrix macros. Keeping them separate allows us to do conditional
  macros that can handle the differences between Clojure and Clojurescript."
  (:require [clojure.core.matrix.macros :refer [TODO c-for]])
  (:import [java.util Arrays]))

(defmacro abutnth [i xs]
  `(let [n# (alength ~xs)
         length# (int (dec n#))
         new-xs# (Arrays/copyOf ~xs length#)]
     (c-for [j# (int ~i) (< j# (dec n#)) (inc j#)]
       (aset new-xs# (int j#) (aget ~xs (int (inc j#)))))
     new-xs#))
