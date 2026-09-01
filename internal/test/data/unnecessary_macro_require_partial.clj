(ns unnecessary-macro-require.partial
  (:require-macros [unnecessary-macro-require.provider :as macros]))

(macros/wrap 1)
