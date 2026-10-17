(ns com.github.ztellman.primitive-math
  (:refer-clojure
    :exclude [* + - / < > <= >= == rem bit-or bit-and bit-xor bit-not bit-shift-left bit-shift-right unsigned-bit-shift-right byte short int float long double inc dec zero? min max true? false?])
  (:import
    [com.github.ztellman.primitive_math Primitives]
    [java.nio ByteBuffer]))

(defmacro bit-shift-left
  "A primitive macro version of `bit-shift-left`."
  [n bits]
  `(Primitives/shiftLeft ~n ~bits))

(defmacro <<
  "An alias for `bit-shift-left`."
  [n bits]
  `(Primitives/shiftLeft ~n ~bits))
