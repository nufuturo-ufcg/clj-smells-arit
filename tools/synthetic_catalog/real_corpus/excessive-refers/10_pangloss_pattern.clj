(ns pattern.nanopass.pass
  (:require [pattern.match.predicator :refer [*pattern-replace*
                                              make-abbr-predicator
                                              with-predicates]]
            [pattern.match.core :refer [compile-pattern]]
            [pattern.nanopass.dialect
             :as d
             :refer [=> ==> ===>
                     =>:from =>:to =>:type
                     dialects
                     from-dialect]]
            [pattern.r3.combinators :as c :refer [on-mutual
                                                  on-subexpressions
                                                  rule-list
                                                  guard
                                                  descend
                                                  directed
                                                  default]]
            [pattern.r3.core :refer [rule success]]
            [pattern.r3.rewrite :refer [quo sub add-env-args*]]))
