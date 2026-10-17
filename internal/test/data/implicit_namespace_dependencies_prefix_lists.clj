(ns implicit-namespace-dependencies-prefix-lists
  (:use [clojush pushstate interpreter]
        [seesaw core]
        [clojure [set :only [union]] [string :only [join]]]
        [clojure.walk :only [postwalk]]
        [mixed [a :only [x]] b]))
;; A prefix list whose entries all carry :only imports nothing implicitly.
(use '[clojure [set :only [union]]])
(use '[seesaw core] 'clojure.data)
