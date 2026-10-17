(ns arcadia.internal.test
  (:require [clojure.test :as test]
            [clojure.spec.alpha :as s]
            [arcadia.internal.macro :as am]
            [arcadia.internal.map-utils :as mu]))

(defn process-is [expr result message]
  {::type ::result
   ::status (if result :pass :fail)
   ::form expr
   ::result result
   ::message message})

(defn process-is-err [expr error message]
  {::type ::result
   ::status :error
   ::form expr
   ::error error
   ::message message})

(defn assert-expr-dispatch [test-expr _]
  (if (seq? test-expr)
    (first test-expr)
    :default))

(defmulti assert-expr #'assert-expr-dispatch)

(defmethod assert-expr :default [test-expr message]
  `(let [expr# (quote ~test-expr)]
     (try
       (process-is expr# ~test-expr ~message)
       (catch Exception e#
         (process-is-err expr# e# ~message)))))

(defn assert-expr-equals [test-expr message]
  (let [expr-arg-bindings (apply concat
                            (for [expr (rest test-expr)]
                              [(gensym "arg_") expr]))]
    `(let [expr# (quote ~test-expr)]
       (try
         (let [~@expr-arg-bindings
               result# (= ~@(take-nth 2 expr-arg-bindings))]
           (if result#
             (process-is expr# result# ~message)
             (assoc (process-is expr# result# ~message)
               ::actual (list '= ~@(take-nth 2 expr-arg-bindings)))))
         (catch Exception e#
           (process-is-err expr# e# ~message))))))

(defmethod assert-expr '= [test-expr message]
  (if (= #'clojure.core/= (ns-resolve *ns* '=))
    (assert-expr-equals test-expr message)
    `(let [expr# (quote ~test-expr)]
       (try
         (process-is expr# ~test-expr ~message)
         (catch Exception e#
           (process-is-err expr# e# ~message))))))

(defmethod assert-expr 'clojure.core/= [test-expr message]
   (assert-expr-equals test-expr message))

(defmacro is
  ([test-expr]
   `(is ~test-expr nil))
  ([test-expr message]
   (assert-expr test-expr message)))

(defmacro throws
  ([test-expr argument-type message]
   `(let [expr# (quote ~test-expr)]
      (try
        (let [res# ~test-expr]
            ;; TODO: should better differentiate in the printout between
            ;; failing `is` and failing `throws`
            {::type ::result 
             ::status :fail
             ::form expr#
             ::result res#
             ::message ~message})
        (catch Exception e#
          (if (instance? ~argument-type e#)
            {::type ::result
             ::status :pass
             ::form expr#
             ::error e#
             ::message ~message}
            {::type ::result
             ::status :error
             ::form expr#
             ::error e#
             ::message ~message}))))))
