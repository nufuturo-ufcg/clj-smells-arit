(ns sci.impl.faster
  {:no-doc true}
  (:require #?@(:cljd [] :default [[sci.impl.macros :refer [?]]]))
  #?(:cljs (:require-macros [sci.impl.faster :refer [nth-2 assoc-3 get-2]])))

#?(:cljd
   (defmacro deref-1 [ref]
     `(deref ~ref))
   :default
   (defmacro deref-1
     [ref]
     (?
      :clj `(.deref ~(with-meta ref
                       {:tag 'clojure.lang.IDeref}))
      :cljs `(~'-deref ~ref))))
