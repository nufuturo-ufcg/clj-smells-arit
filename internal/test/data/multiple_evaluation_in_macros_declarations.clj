(ns multiple-evaluation-in-macros-declarations
  (:require [example.monad :refer [mlet return]]))

;; A defmethod parameter vector declares locals; it evaluates nothing.
(defmacro defread [args body]
  `(do (defmethod read-a :k ~args ~body)
       (defmethod read-b :k ~args nil)))

;; A set! place names a field; only the assigned value is evaluated.
(defmacro caching [coll hash-key]
  `(let [h# ~hash-key]
     (if (zero? h#) (set! ~hash-key (hash ~coll)) h#)))

;; An unknown form with the parameter in a binding position may bind it.
(defmacro bind-then-return [expr name]
  `(mlet [~name ~expr] (return ~name)))

;; Evaluated twice on one path: still reported.
(defmacro twice-in-method [k x]
  `(defmethod show ~k [_#] (str ~x ~x)))

(defmacro set-twice [x]
  `(set! *flag* (and ~x ~x)))

;; Forwarding to an arity that only quotes the argument evaluates it nowhere.
(defmacro validate
  ([opts src] `(validate ~opts ~src ~src))
  ([opts src alt-name] `(check ~opts (name (quote ~alt-name)) '~src)))

;; A body spliced into a DSL macro of another namespace may be compiled to data.
(defmacro local-script [& body]
  `(run (stevedore/script ~@body) (stevedore/script ~@body)))
