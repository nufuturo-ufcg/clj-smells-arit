(ns require-macros
  #?(:clj (:require [required-macros :refer [do2]]))
  #?(:cljs (:require-macros [require-macros :refer [add subtract]]
                            [required-macros :refer [do2]])))

(defmacro add
  [a b]
  `(+ ~a ~b))
