(ns unnecessary-macro-require.provider)

(defmacro wrap
  ([x] `(+ ~x 1))
  ([x y] `(+ ~x ~y)))
