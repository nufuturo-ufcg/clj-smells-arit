(ns namespace-load-side-effects
  (:require [clojure.string :as str]
            [clojure.test :refer [deftest]]))

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Direct top-level require call outside ns
(require '[clojure.data :as data])

;; Example 2: Direct top-level requiring-resolve call outside ns
(requiring-resolve 'cheshire.core/parse-string)

;; Example 3: Top-level require execution inside a global 'def' binding
(def _load-utils
  (require '[clojure.walk :as walk]))

;; Example 4: Top-level requiring-resolve assigned to a var at load-time
(def parse-json-fn
  (requiring-resolve 'cheshire.core/parse-string))

;; Example 5: Top-level require inside a conditional 'when' guard
(when true
  (require '[clojure.pprint :as pp]))

;; Example 6: Top-level require inside a conditional 'if' branch
(if (System/getProperty "debug")
  (require '[clojure.stacktrace :as st])
  (println "Debug inativo"))

;; Example 7: Top-level require inside a standalone/unhandled 'try-catch' block (False Negative)
(try
  (require '[com.meu-app.opcional :as opt])
  (catch Exception _ nil))

;; Example 8: Top-level require inside a 'let' binding
(let [lib-name 'clojure.zip]
  (require lib-name))

;; Example 9: Hidden top-level load via function alias / var redirection (False Negative)
(def carregar-modulo require)
(carregar-modulo '[com.meu-app.oculto :as oculto])

;; Example 10: Top-level require executed inside 'defonce' initialization
(defonce _init-deps
  (do (require '[com.meu-app.db :as db]) :pronto))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Standard clean static dependency declaration inside ns macro
(defn format-text [input-str]
  (str/trim (str/lower-case input-str)))

;; Example 12: Legitimate lazy runtime require inside a 'defn' function body (False Positive)
(defn load-plugin-on-demand [plugin-name]
  (let [ns-sym (symbol plugin-name)]
    (require ns-sym)
    ((requiring-resolve (symbol (str ns-sym "/start"))))))

;; Example 13: Deferred require inside a test definition 'deftest' (False Positive)
(deftest integration-test-helper
  (require '[com.meu-app.test-utils :as tu]))

;; Example 14: Deferred requiring-resolve inside an anonymous function 'fn' (False Positive)
(def get-lazy-mapper
  (fn [] ((requiring-resolve 'cheshire.core/parse-string) "{}")))

;; Example 15: Deferred require inside a 'defmethod' multimethod body (False Positive)
#_{:clj-kondo/ignore [:unresolved-symbol]}
(defmethod process-event :pdf [evt]
  (require '[com.meu-app.pdf-generator :as pdf]))

;; Example 16: Deferred require enclosed in a 'delay' macro (runs only on @/deref) (False Positive)
(def lazy-data-xml
  (delay (require '[clojure.data.xml :as xml])))

;; Example 17: Deferred require inside an asynchronous 'future' block (False Positive)
(defn init-background-services []
  (future (require '[com.meu-app.bg-worker :as bg])))

;; Example 18: Quoted require form treated as static AST data (False Positive)
(def macro-ast-template
  '(require '[clojure.java.io :as io]))

;; Example 19: Ignored require inside Rich Comment Block (False Positive)
(comment
  (require '[clojure.tools.namespace.repl :refer [refresh]]))

;; Example 20: Runtime conditional requiring-resolve guarded inside function with System/getenv
(defn fetch-telemetry []
  (when (System/getenv "ENABLE_TELEMETRY")
    ((requiring-resolve 'com.meu-app.telemetry/collect))))