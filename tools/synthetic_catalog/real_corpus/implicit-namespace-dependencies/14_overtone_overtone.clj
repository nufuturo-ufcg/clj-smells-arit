(ns
    ^{:doc "Local asset registry mechanism. Maintains a live file-store which maps asset 'keys' to local files."
      :author "Kevin Neaton"}
   overtone.libs.asset.store
  (:use [clojure.java.io :only [file]]
        [clojure.walk :only [postwalk]]
        [overtone.helpers file]
        [overtone.config.store :only [OVERTONE-ASSETS-FILE]]
        [overtone.config.file-store]))
