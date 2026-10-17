#!/usr/bin/env bb

(ns graal-tests
  (:require
   [clojure.string :as str]
   [babashka.fs    :as fs]
   [babashka.process :refer [shell]]))
