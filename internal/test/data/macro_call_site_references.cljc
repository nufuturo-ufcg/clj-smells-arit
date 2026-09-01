(ns macro-call-site.references
  (:require [macro-call-site.provider :refer [duplicated]])
  (:require-macros [macro-call-site.provider :as macros]))

(duplicated 42)
(macros/duplicated 43)
