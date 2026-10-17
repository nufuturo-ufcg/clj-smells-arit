(ns ^:no-doc zprint.main
  (:require ;[clojure.string :as str]
    [zprint.config :refer [config-and-validate-all get-explained-options
                           get-explained-set-options get-options merge-deep
                           sci-load-string select-op-options vec-str-to-str]]
    [zprint.core   :refer [czprint load-options! set-options! zprint-file-str
                           zprint-str]]
    #?@(:clj [[babashka.fs :refer [glob]]]))
  #?(:clj (:gen-class)))

#?(:clj (defn write-to-stderr
          "Take a string, and write it to stderr."
          [s]
          (let [^java.io.Writer w (clojure.java.io/writer *err*)]
            (.write w (str s "\n"))
            (.flush w))))
