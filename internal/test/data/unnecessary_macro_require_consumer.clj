(ns unnecessary-macro-require.consumer
  (:require-macros [unnecessary-macro-require.provider :as macros]))

(macros/wrap 1)
(macros/wrap 1 2)
