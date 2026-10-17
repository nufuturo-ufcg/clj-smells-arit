(ns relying-on-load-time-platform)

#?(:bb (defn edn-resource [file] (slurp file))
   :clj (defmacro edn-resource [file] `(quote ~file)))

(def dictionary (edn-resource "dicts/en.edn"))
