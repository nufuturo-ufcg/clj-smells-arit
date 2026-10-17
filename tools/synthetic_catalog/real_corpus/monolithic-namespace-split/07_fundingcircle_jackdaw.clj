(ns jackdaw.data
  "This namespace aims to provide `->T`, `(datafy T)`, and `data->T` as
  a round-tripping of Katka's (client) record types.

  Note that for some types, particularly Kafka's `-Result` types no
  `->T` constructors are provided as there are no consumers within the
  Kafka API for these records they are merely packed results.

  For compatibility with Clojure before 1.10.0, a `datafy` function is
  provided. On 1.10 or after, it simply defers to
  `clojure.datafy/datafy` but before 1.10 it acts as a backport
  thereof.

  "
  {:license "BSD 3-Clause License <https://github.com/FundingCircle/jackdaw/blob/master/LICENSE>"}
  (:require [clojure.walk :refer [stringify-keys keywordize-keys]])
  (:import java.util.Properties))

(load "/jackdaw/data/common")
