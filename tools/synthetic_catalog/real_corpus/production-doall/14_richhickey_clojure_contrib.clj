(ns
  #^{:author "Konrad Hinsen"
     :doc "Local macros and symbol macros

           Local macros are defined by a macrolet form. They are usable only
           inside its body. Symbol macros can be defined globally
           (defsymbolmacro) or locally (symbol-macrolet). A symbol
           macro defines a form that replaces a symbol during macro
           expansion. Function arguments and symbols bound in let
           forms are not subject to symbol macro expansion.

           Local macros are most useful in the definition of the expansion
           of another macro, they may be used anywhere. Global symbol
           macros can be used only inside a with-symbol-macros form."}
  clojure.contrib.macro-utils
  (:use [clojure.contrib.def :only (defvar-)]))

(defn- reserved?
  [symbol]
  "Return true if symbol is a reserved symbol (starting or ending with a dot)."
  (let [s (str symbol)]
    (or (= "." (subs s 0 1))
	(= "." (subs s (dec (count s)))))))

(defn- expand-symbol
  "Expand symbol macros"
  [symbol]
  (cond (contains? protected-symbols symbol) symbol
	(reserved? symbol)                   symbol
	(contains? macro-symbols symbol)     (get macro-symbols symbol)
	:else (let [v (resolve symbol)
		    m (meta v)]
		(if (:symbol-macro m)
		  (var-get v)
		  symbol))))

(defn- expand-1
  "Perform a single non-recursive macro expansion of form."
  [form]
  (cond
    (seq? form)
      (let [f (first form)]
        (cond (contains? special-forms f) form
	      (contains? macro-fns f)     (apply (get macro-fns f) (rest form))
	      (symbol? f)                 (let [exp (expand-symbol f)]
					    (if (= exp f)
					      (clojure.core/macroexpand-1 form)
					      (cons exp (rest form))))
	      ; handle defmacro macros and Java method special forms
	      :else (clojure.core/macroexpand-1 form)))
    (symbol? form)
      (expand-symbol form)
     :else
       form))

(defn- expand
  "Perform repeated non-recursive macro expansion of form, until it no
   longer changes."
  [form]
  (let [ex (expand-1 form)]
    (if (identical? ex form)
      form
      (recur ex))))

(defn- expand-args
  "Recursively expand the arguments of form, leaving its first
   n elements unchanged."
  ([form]
   (expand-args form 1))
  ([form n]
   (doall (concat (take n form) (map expand-all (drop n form))))))

(defn- expand-fn-body
  [[args & exprs]]
  (binding [protected-symbols (reduce conj protected-symbols
				     (filter #(not (= % '&)) args))]
    (cons args (doall (map expand-all exprs)))))

(defn- expand-list
  "Recursively expand a form that is a list or a cons."
  [form]
  (let [f (first form)]
    (if (symbol? f)
      (if (contains? special-forms f)
	((get special-form-handlers f expand-args) form)
	(expand-args form))
      (doall (map expand-all form)))))

(defn- expand-all
  "Expand a form recursively."
  [form]
  (let [exp (expand form)]
    (cond (symbol? exp) exp
	  (seq? exp) (expand-list exp)
	  (vector? exp) (into [] (map expand-all exp))
	  (map? exp) (into {} (map expand-all (seq exp)))
	  :else exp)))
