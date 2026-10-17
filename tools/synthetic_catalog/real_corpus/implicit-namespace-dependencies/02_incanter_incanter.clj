(ns incanter.sql
  "SQL module for interacting with databases."
  (:require [clojure.core.matrix.dataset :as ds])
  (:use [incanter.core :only [dataset]]
        [clojureql.core :as ql :exclude [conj! disj! take drop distinct case compile sort]]))
