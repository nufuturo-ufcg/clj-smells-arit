(ns ^:no-doc zprint.sutil
  (:require clojure.string
            #?@(:bb []
                :clj [[zprint.macros :refer [do-redef-vars]]])
            zprint.zfns
            #?@(:clj [[zprint.redef]])
            #?@(:cljs [[cljs.reader :refer [read-string]]])))

#?(:clj
     (do
       (def hexseq
         ["0" "1" "2" "3" "4" "5" "6" "7" "8" "9" "a" "b" "c" "d" "e" "f"])
       (def hexbyte
         (vec (map #(str (nth hexseq (bit-shift-right (bit-and % 240) 4))
                         (nth hexseq (bit-and % 15)))
                (range 256))))
       (defn hexadj [b] (if (< b 0) (+ b 256) b))
       (defn hex-byte
         "Turn the low byte of a number into hex"
         [n]
         (nth hexbyte (hexadj (bit-and n 255))))
       (defn hex-number
         "Turn a number into hex. The shift-seq encodes the amount of the number
  that should be turned into hex."
         [n shift-seq]
         (apply str (map #(hex-byte (unsigned-bit-shift-right n %)) shift-seq)))
       (def int-array-type (type (int-array [0])))
       (def byte-array-type (type (byte-array [0])))
       (def short-array-type (type (short-array [0])))
       (def long-array-type (type (long-array [0])))
       (defn array-to-shift-seq
         "Given an array of integers, what is the shift-seq to give
  to hex-number to make them into hex?"
         [a]
         (let [t (type a)]
           (cond (= t byte-array-type) [0]
                 (= t short-array-type) [8 0]
                 (= t int-array-type) [24 16 8 0]
                 (= t long-array-type) [56 48 40 32 24 16 8 0]
                 :else nil)))
       (defn snumstr
         "Does pr-str, but takes an additional argument for hex conversion. Only
  works for bytes at this time."
         [zloc hex? shift-seq]
         (if (and (integer? zloc) hex?)
           (if (string? hex?)
             (str hex? (hex-number zloc shift-seq))
             (hex-number zloc shift-seq))
           (pr-str zloc))))
   :cljs (defn snumstr "Does pr-str." [zloc hex? shift-seq] (pr-str zloc)))
