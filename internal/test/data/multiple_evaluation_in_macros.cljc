(ns multiple-evaluation-in-macros-cljc)

(defmacro duplicated-cljc [expr]
  `(do ~expr ~expr))

(defmacro compile-branch-safe-cljc [expr]
  `(if-cljs ~expr ~expr))
