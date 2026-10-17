(ns clojush.problems.synthetic.string-generation
  (:use [clojush.pushgp.pushgp]
        [clojush util pushstate interpreter random])
  (:require [clojure.java.io :as io]))

(def spit-improvement-vector? false)

(def output_folder "/home/juao15/autoconstruction/string-generation/improvement_logs")

(def log-file
  (if spit-improvement-vector?
    (loop [i 0]
      (let [file (io/file output_folder (str i ".log"))]
        (if (.exists file)
          (recur (inc i))
          (do
            (spit file [])
            file))))))
