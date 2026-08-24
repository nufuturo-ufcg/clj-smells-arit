(ns namespace-load-side-effects-fn-literal)

(def mapper
  #(requiring-resolve 'example.runtime/parse))
