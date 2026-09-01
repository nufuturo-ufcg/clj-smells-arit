(ns unnecessary-macro-require.indirect
  (:require-macros [unnecessary-macro-require.provider :as macros]))

(let [macro-fn macros/wrap]
  (macro-fn 1))
