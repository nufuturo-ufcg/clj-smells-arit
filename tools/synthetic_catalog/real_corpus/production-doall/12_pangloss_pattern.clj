(ns pattern.r3.core
  "This namespace bootstraps the core rule macro and some utility functions."
  (:require [pattern.match.core :refer [pattern-names compile-pattern]]
            pattern.matchers
            [genera.trampoline :refer [bounce?]]
            [pattern.substitute :refer [substitute]]
            [pattern.match.predicator :refer [*pattern-replace* apply-replacements]]
            [clojure.walk :as walk]
            [pattern.types :refer [->SuccessUnmodified ->Success ->SuccessEnv]]
            [pattern.r3.rule :refer [->Rule *post-processor* *identity-rule-post-processor*
                                     make-rule -rebuild-rule]])
  (:import (pattern.types Success SuccessEnv SuccessUnmodified)
           (clojure.lang IFn IObj IMeta)))

(defn- extract-env-args [env-args]
  (when (seq env-args)
    (doall
      (mapcat (fn [arg]
                [(symbol (name arg)) `(~'%env ~(keyword arg))])
        env-args))))
