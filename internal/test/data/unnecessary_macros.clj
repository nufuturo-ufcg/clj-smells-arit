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

(defmacro documented "A documented wrapper" [x]
  `(+ ~x 1))

(defmacro attributed {:private true} [x]
  `(+ ~x 1))

(defmacro quoted-symbol-after-unquote [x y]
  `(list ~x y))

(defmacro multiple-body-forms [x]
  (println "expanded")
  `(+ ~x 1))

(let [defmacro (fn [& _] nil)]
  (defmacro shadowed [x]
    `(+ ~x 1)))

;; Primitive Java interop is part of the macro's generated calling convention.
(defmacro primitive-wrapper [x]
  `(.invokePrim ~x))

;; Corpus-derived negatives: each shape below cannot become a function.
(defmacro defining [n x]
  `(defn ~n [] ~x))

(defmacro comprehension [bindings body]
  `(doall (for ~bindings ~body)))

(defmacro quoted-symbol [sym]
  `(requiring-resolve '~sym))

(defmacro thunked [body]
  `(with-header* #(do ~body)))

(defmacro captured-this [sym new-val]
  `(swap! @~'this assoc ~sym ~new-val))

(defmacro delegates-to-macro [expr]
  `(format "Elapsed: %f" (:time (time ~expr))))

(defmacro expansion-time-work [tokens]
  `(str ~(count tokens)))

(defmacro unresolved-helper [src]
  `(error! "Unsupported: %s" ~src))

;; Positives: plain runtime calls to known functions.
(defn local-helper [x] (inc x))

(defmacro calls-local-function [x]
  `(local-helper ~x))

(defmacro core-composition [m]
  `(not (empty? ~m)))
