(ns unnecessary-macros)

(defmacro add-one [x]
  `(+ ~x 1))

(defmacro pair [a b]
  `[~a ~b])

(defmacro repeated-evaluation [x]
  `(+ ~x ~x))

(defmacro compile-time-name [x]
  (let [g (gensym "value")]
    `(let [~g ~x] ~g)))

(defmacro map-with-semantic-nil [x]
  `{:value ~x :error nil})
