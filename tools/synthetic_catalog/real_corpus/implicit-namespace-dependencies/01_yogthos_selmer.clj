(ns selmer.parser
  " Parsing and handling of compile-time vs.
  run-time. Avoiding unnecessary work by pre-processing
  the template structure and content and reacting to
  the runtime context map with a prepared data structure
  instead of a raw template. Anything other than a raw tag
  value injection is a runtime dispatch fn. Compile-time here
  means the first time we see a template *at runtime*, not the
  implementation's compile-time. "
  (:require
   [clojure.set :as set]
   [clojure.string :as string]
   [selmer.template-parser :refer [preprocess-template]]
   [selmer.filters :refer [filters]]
   [selmer.filter-parser :refer [compile-filter-body literal?
                                 split-value parse-literal]]
   [selmer.tags :refer :all]
   [selmer.util :refer :all]
   [selmer.validator :refer [validation-error]]
   selmer.node)
  (:import [selmer.node TextNode FunctionNode]))
