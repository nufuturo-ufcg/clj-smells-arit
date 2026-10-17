(ns tupelo.core
  "Tupelo - Making Clojure even sweeter"
  ; We use the self-require trick to force separate compilation stages for macros
  ; See "ClojureScript Macro Tower & Loop" by Mike Fikes (2015-12-18)
  ;   https://code.thheller.com/blog/shadow-cljs/2019/10/12/clojurescript-macros.html
  ;   http://blog.fikesfarm.com/posts/2015-12-18-clojurescript-macro-tower-and-loop.html
  #?(:cljs (:require-macros
             [tupelo.core.impl]
             [tupelo.core :refer [it-> cond-it-> some-it->
                                  vals->map with-map-vals forv
                                  with-spy-indent spyx spyxx spy-pretty spyx-pretty
                                  let-spy let-spy-pretty let-some map-let* map-let lazy-cons
                                  assert-info try-catchall with-exception-default verify-form
                                  if-java-1-8-plus
                                  when-clojure-1-8-plus when-clojure-1-9-plus when-not-clojure-1-9-plus
                                  destruct lazy-gen yield yield-all matches?]]))

  (:require
    [clojure.core :as cc]
    [clojure.core.async :as async]
    [clojure.data.avl :as avl]
    [clojure.pprint :as pprint]
    [clojure.set :as set]
    [clojure.string :as str]
    [clojure.walk :as walk]
    [schema.core :as s]
    [tupelo.core.impl :as impl]
    [tupelo.lexical :as lex]
    [tupelo.schema :as tsk])
  #?(:clj (:require [cheshire.core :as cheshire]
                    [clojure.core.match :as ccm]
                    [flatland.ordered.map :as omap]
                    [flatland.ordered.set :as oset]
                    [tupelo.types :as types]))
  #?(:clj (:import [java.io BufferedReader ByteArrayOutputStream PrintStream StringReader OutputStream]
                   [java.nio ByteBuffer])))

(defn cljs-env?     ; from plumatic schema/macros.clj
  "Given the &env from a macro, and tell whether we are expanding into cljs."
  [env] (boolean (:ns env)))

(defmacro if-cljs   ; from plumatic schema/macros.clj
  "For use within macros. Return `then-form` if we are generating ClojureScript code,
   and `else-form` for Clojure code.
     (from https://groups.google.com/d/msg/clojurescript/iBY5HaQda4A/w1lAQi9_AwsJ)"
  [then-form else-form]
  (if (cljs-env? &env) then-form else-form))

(defn try-catchall-impl ; from plumatic schema/macros.clj
  "A cross-platform variant of try-catch that catches all exceptions.
   Does not (yet) support finally, and does not need or want an exception class."
  [forms]
  (let [try-body (butlast forms)
        [catch-op ex-symbol & catch-body] (last forms)]
    (assert (= catch-op 'catch))
    (assert (symbol? ex-symbol))
    `(tupelo.core/if-cljs
       (try ~@try-body (catch js/Object ~ex-symbol ~@catch-body))
       (try ~@try-body (catch Throwable ~ex-symbol ~@catch-body)))))

(defmacro try-catchall ; from plumatic schema/macros.clj
  "A cross-platform variant of try-catch that catches all exceptions.
   Does not (yet) support finally, and does not need or want an exception class."
  [& forms] (try-catchall-impl forms))

(defmacro with-exception-default
  "Evaluates body & returns its result.  In the event of an exception, default-val is returned
   instead of the exception."
  [default-val & forms] ; :default
  `(try-catchall
     ~@forms
     (catch e# ~default-val)))

(defn type-name-str
  "Returns the type/class name of a value as a string.  Works for both CLJ and CLJS."
  [arg] (tupelo.core.impl/type-name-str arg))

(defn assert-info-impl
  [[pred-expr err-str info-expr]]
  `(when-not ~pred-expr
     (throw (ex-info ~err-str ~info-expr))))

(defmacro assert-info ; #todo keep or discard???  Needs CLJS macro support
  "Like `assert` but accepts a map like `ex-info`:

        (assert-info (sequential? listy)
          \"prepend: Sequential collection required, found=\"
          {:listy listy} )

   Implemented using `(throw (ex-info ...))` "
  [pred-expr err-str info-expr] (assert-info-impl [pred-expr err-str info-expr]))

(defn truthy?
  "Returns true if arg is logical true (neither nil nor false); otherwise returns false."
  [arg] (if arg true false))

(defn nl
  "Abbreviated name for `newline`.  Accepts varargs to be printed 1 per line after initial newline. "
  [& args]
  (newline)
  (doseq [arg args]
    (println arg)))

(s/defn kw->sym :- s/Symbol
  "Converts a keyword to a symbol"
  [arg :- s/Keyword]
  (symbol (name arg)))

(s/defn not-nil? :- s/Bool
  "Returns true if arg is not nil; false otherwise. Equivalent to (not (nil? arg)),
   or the poorly-named clojure.core/some? "
  [arg :- s/Any]
  (not (nil? arg)))

(s/defn not-empty? :- s/Bool
  "For any collection coll, returns true if coll contains any items; otherwise returns false.
   Equivalent to (not (empty? coll))."
  ; [coll :- [s/Any]]  ; #todo extend Prismatic Schema to accept this for strings
  [coll]
  (not (cc/empty? coll)))

(defn sorted-map-generic
  "Returns a generic sorted map, able to accept keys of different classes"
  [] (sorted-map-by lex/compare-generic))

(defn sorted-set-generic
  "Returns a generic sorted set, able to accept keys of different classes"
  [] (sorted-set-by lex/compare-generic))

(defn walk-data->pretty
  "Recursively walks a data structure, converting all maps & sets to (generic) sorted versions,
   and all sequences to vectors. "
  [data]
  (let [pretty-item (fn pretty-item-fn
                      [item]
                      ;(spyx :awt--607 (type item))
                      ;(spyx :awt--607 (class item))
                      (cond
                        (sequential? item) (vec item)

                        #?@(:clj  [(map? item) (into (sorted-map-generic) item)
                                   (set? item) (into (sorted-set-generic) item)

                                   ;(comment ; #todo #awt 22-7-05 fix this!
                                   ;  (atom? item) item
                                   ;  (= datomic.db.Datum (type item)) item
                                   ;  )
                                   ]
                            :cljs [(map? item) (into (sorted-map) item) ; #todo => (sorted-map-generic)
                                   (set? item) (into (sorted-set) item) ; #todo => (sorted-map-generic)
                                   ])

                        :else item))
        result      (walk/postwalk pretty-item data)]
    result))

(defn unlazy        ; #todo need tests & docs. Use for datomic Entity?
  "Converts a lazy collection to a concrete (eager) collection of the same type."
  [coll]
  (let [unlazy-item (fn unlazy-item-fn
                      [item]
                      ;(newline)
                      ;(prn :awt--unlazy-item-fn---------------------------------------------------------------------------)
                      ;(prn "(type item) => " (type item))
                      ;(prn "item => " item)
                      (with-exception-default item
                        (cond
                          (sequential? item) (vec item)
                          (map? item) (into {} item)
                          (set? item) (into #{} item)

                          #?@(:clj [
                                    (instance? java.io.InputStream item) (slurp item) ; #todo need test
                                    (instance? java.util.List item) (vec item) ; #todo need test
                                    (instance? java.util.Map item) (into {} item) ; #todo need test
                                    (instance? java.util.Set item) (into #{} item) ; #todo need test
                                    (instance? java.lang.Iterable item) (into [] item) ; #todo need test
                                    (= "datomic.query.EntityMap" (type-name-str item)) (into {} item) ; type-free test (doesn't need datomic)
                                    ])
                          :else item)))
        result      (walk/prewalk unlazy-item coll)]
    result))

(defn unlazy-pretty
  "Shorthand for (walk-data->pretty (unlazy data))."
  [data]
  #?(:clj  (try
             (walk-data->pretty (unlazy data))
             (catch Throwable t
               (prn :*****************************************************************************)
               (prn :**********--error--**********--tupelo.core/unlazy-pretty--Throwable--********)
               (prn :*****************************************************************************)
               (.printStackTrace t)
               data))
     :cljs (walk-data->pretty (unlazy data))
     ))

(defn has-length?   ; #todo rework => (count= N coll)  ???
  "Returns true if the collection has the indicated length. Does not hang for infinite sequences."
  [coll n]
  (assert-info (not-nil? coll) "has-length?: coll must not be nil" {:coll coll})
  (let [take-items (cc/take n coll)
        rest-items (cc/drop n coll)]
    (and (= n (count take-items))
         (empty? rest-items))))

(defn only
  "Ensures that a sequence is of length=1, and returns the only value present.
  Throws an exception if the length of the sequence is not one.
  Note that, for a length-1 sequence S, (first S), (last S) and (only S) are equivalent."
  [coll]
  (assert-info (has-length? coll 1) "only: num-items must=1" {:coll coll})
  (clojure.core/first coll))

(defn xfirst        ; #todo -> tests
  "Returns the first value in a list or vector. Throws if empty."
  [coll]
  (assert-info (and (not-nil? coll) (not-empty? coll))
               "xfirst: invalid coll: " {:coll coll})
  (nth coll 0))

(defn xsecond       ; #todo -> tests
  "Returns the second value in a list or vector. Throws if (< len 2)."
  [coll]
  (assert-info (and (not-nil? coll) (not-empty? coll))
               "xsecond: invalid coll: " {:coll coll})
  (nth coll 1))

(s/defn xlast :- s/Any ; #todo -> tests
  "Returns the last value in a list or vector. Throws if empty."
  [coll :- [s/Any]]
  (assert-info (and (not-nil? coll) (not-empty? coll)) "xlast: invalid coll: " {:coll coll})
  (clojure.core/last coll))

(defn xrest         ; #todo -> tests
  "Returns the last value in a list or vector. Throws if empty."
  [coll]
  (assert-info (and (not-nil? coll) (not-empty? coll)) "xrest: invalid coll: " {:coll coll})
  (clojure.core/rest coll))

(defmacro forv      ; #todo rename for-vec ???
  "Like clojure.core/for but returns results in a vector.
  Wraps the loop body in a `do` as with `doseq`. Not lazy."
  [& forms]
  (let [bindings-vec (xfirst forms)
        body-forms   (xrest forms)]
    `(vec (for ~bindings-vec
            (do ~@body-forms)))))

(defn keep-if
  "Returns a vector of items in coll for which (pred item) is true (alias for clojure.core/filter)"
  [pred coll]
  (cond
    (sequential? coll) (vec (clojure.core/filter pred coll))
    (map? coll) (reduce-kv (fn [cum-map k v]
                             (if (pred k v)
                               (assoc cum-map k v)
                               cum-map))
                           {}
                           coll)
    (set? coll) (reduce (fn [cum-set elem]
                          (if (pred elem)
                            (conj cum-set elem)
                            cum-set))
                        #{}
                        (seq coll))
    :else (throw (ex-info "keep-if: coll must be sequential, map, or set." {:coll coll}))))

(defn drop-if
  "Returns a vector of items in coll for which (pred item) is false (alias for clojure.core/remove)"
  [pred coll]
  (keep-if (complement pred) coll))

(s/defn append :- tsk/List
  "Given a sequential object (vector or list), add one or more elements to the end."
  [listy :- tsk/List
   & elems :- [s/Any]]
  (assert-info (sequential? listy) "append: Sequential collection required, found=" {:listy listy})
  (assert-info (not-empty? elems) "Nothing to append! elems=" {:elems elems})
  ; (vec (concat listy elems))  ; #TODO ***WARNING*** never use `concat`!!! 1000x slower!
  (into (vec listy) elems))

(s/defn prepend :- tsk/List
  "Given a sequential object (vector or list), add one or more elements to the beginning"
  [& args]
  (let [elems (butlast args)
        listy (xlast args)]
    (assert-info (sequential? listy) "prepend: Sequential collection required, found=" {:listy listy})
    (assert-info (not-empty? elems) "Nothing to prepend! elems=" {:elems elems})
    (into (vec elems) listy)))

(def ^:dynamic *spy-enabled* true)

(def ^:no-doc spy-indent-level (atom 0))

(defn ^:no-doc spy-indent-spaces []
  (str/join (repeat (* 2 @spy-indent-level) \space)))

(defn ^:no-doc spy-indent-inc
  "Increase the spy indent level by one."
  []
  (swap! spy-indent-level inc))

(defn ^:no-doc spy-indent-dec
  "Decrease the spy indent level by one."
  []
  (swap! spy-indent-level dec))

(defn ^:no-doc spy2-impl ; 2-arg arity requires user-supplied keyword
  [arg1 arg2]
  (let [[tag value] (cond
                      (keyword? arg1) [arg1 arg2]
                      (keyword? arg2) [arg2 arg1]
                      :else (throw (ex-info "spy: either first or 2nd arg must be a keyword tag \n   args:"
                                            {:arg1 arg1
                                             :arg2 arg2})))]
    (when *spy-enabled*
      (println (str (spy-indent-spaces) tag " => " (pr-str value))))
    value))

(defmacro spy       ; #todo clean up all spy stuff
  "A form of (println ...) to ease debugging display of either intermediate values in threading
   forms or function return values. There are three variants.  Usage:

    (spy :msg <msg-string>)

      This variant is intended for use in either thread-first (->) or thread-last (->>)
      forms.  The keyword :msg is used to identify the message string and works equally
      well for both the -> and ->> operators. Spy prints both <msg-string>  and the
      threading value to stdout, then returns the value for further propogation in the
      threading form. For example, both of the following:

            (->   2
                  (+ 3)
                  (spy :msg \"sum\" )
                  (* 4))
            (->>  2
                  (+ 3)
                  (spy :msg \"sum\" )
                  (* 4))

        will print 'sum => 5' to stdout.

    (spy <msg-string> <value>)
      This variant is intended for simpler use cases such as function return values.
      Function return value expressions often invoke other functions and cannot be
      easily displayed since (println ...) swallows the return value and returns nil
      itself.  Spy will output both <msg-string> and the value, then return the value
      for use by further processing.  For example, the following:

            (println (* 2
                       (spy \"sum\" (+ 3 4))))
      will print:

            sum => 7
            14

      to stdout.

    (spy <value>)
        This variant is intended for use in very simple situations and is the same as the
        2-argument arity where <msg-string> defaults to 'spy'.  For example (spy (+ 2 3))
        prints 'spy => 5' to stdout.  "
  ; 1-arg arity uses a generic "spy" message
  ([value]
   (let [src-line   (:line (meta &form))
         src-ns-str (str (ns-name *ns*))
         out-tag    (keyword (format "spy--%s--line-%03d" src-ns-str src-line))]
     `(let [result# ~value]
        (println ~out-tag "=>" (pr-str result#))
        result#)))
  ([arg1 arg2]
   `(spy2-impl ~arg1 ~arg2)))

(defn ^:no-doc spyx-impl
  [exprs]
  (let [r1         (for [expr (butlast exprs)]
                     (when *spy-enabled*
                       (if (keyword? expr)
                         `(when *spy-enabled* (print (str (spy-indent-spaces) ~expr \space)))
                         `(when *spy-enabled* (println (str (spy-indent-spaces) '~expr " => " ~expr))))))
        r2         (let [expr (xlast exprs)]
                     `(let [spy-val# ~expr]
                        (when *spy-enabled*
                          (println (str (spy-indent-spaces) '~expr " => " (pr-str spy-val#))))
                        spy-val#))
        final-code `(do ~@r1 ~r2)]
    final-code))

(defmacro spyx
  "An expression (println ...) for use in threading forms (& elsewhere). Evaluates the supplied
   expressions, printing both the expression and its value to stdout. Returns the value of the
   last expression."
  [& exprs]
  (spyx-impl exprs))

(defn ^:no-doc spy-pretty-impl
  [exprs]
  (let [r1         (for [expr (butlast exprs)]
                     `(when *spy-enabled* (println (spy-indent-spaces) (str ~expr " => "))))
        r2         (let [expr (xlast exprs)]
                     `(let [spy-val# ~expr]
                        (when *spy-enabled*
                          (println (impl/indent-lines-with (spy-indent-spaces)
                                                           (pretty-str spy-val#))))
                        spy-val#))
        final-code `(do
                      ~@r1
                      ~r2)]
    final-code))

(defmacro spy-pretty ; #todo => core
  "Like `spyx-pretty` but without printing the original form"
  [& exprs]
  (spy-pretty-impl exprs))

(defn ^:no-doc spyx-pretty-impl
  [exprs]
  (let [r1         (for [expr (butlast exprs)]
                     (if (keyword? expr)
                       `(when *spy-enabled* (println (spy-indent-spaces) (str ~expr)))
                       `(when *spy-enabled* (println (spy-indent-spaces) (str '~expr " => " ~expr)))))
        r2         (let [expr (xlast exprs)]
                     `(let [spy-val# ~expr]
                        (when *spy-enabled*
                          (println (str (spy-indent-spaces) '~expr " => "))
                          (println (impl/indent-lines-with (spy-indent-spaces)
                                                           (pretty-str spy-val#))))
                        spy-val#))
        final-code `(do
                      ~@r1
                      ~r2)]
    final-code))

(defmacro spyx-pretty
  "Like `spyx` but with pretty printing (clojure.pprint/pprint)"
  [& exprs]
  (spyx-pretty-impl exprs))

(defmacro with-spy-indent
  "Increments indentation level of all spy, spyx, or spyxx expressions within the body."
  [& forms]
  `(try
     (spy-indent-inc)
     (do ~@forms)
     (finally       ; ensure we un-do indentation in event of exception
       (spy-indent-dec))))

(defmacro let-spy
  "An expression (println ...) for use in threading forms (& elsewhere). Evaluates the supplied
   expressions, printing both the expression and its value to stdout. Returns the value of the
   last expression."
  [& exprs]
  (let [decls      (xfirst exprs)
        _          (assert-info (even? (count decls)) "spy-let-proc: uneven number of decls:" {:decls decls})
        forms      (xrest exprs)
        fmt-pair   (fn [[dest src]]
                     [dest src
                      '_ (list `spyx dest)]) ; #todo gensym instead of underscore?
        pairs      (vec (partition 2 decls))
        r1         (vec (mapcat fmt-pair pairs))
        final-code `(let ~r1 ~@forms)]
    final-code))

(defmacro let-spy-pretty ; #todo -> deprecated
  "An expression (println ...) for use in threading forms (& elsewhere). Evaluates the supplied
   expressions, printing both the expression and its value to stdout. Returns the value of the
   last expression."
  [& exprs]
  (let [decls      (xfirst exprs)
        _          (assert-info (even? (count decls)) "spy-let-pretty-impl: uneven number of decls:" {:decls decls})
        forms      (xrest exprs)
        fmt-pair   (fn [[dest src]]
                     [dest src
                      '_ (list `spyx-pretty dest)]) ; #todo gensym instead of underscore?
        pairs      (vec (partition 2 decls))
        r1         (vec (mapcat fmt-pair pairs))
        final-code `(let ~r1 ~@forms)]
    final-code))

(defmacro spyxx
  "An expression (println ...) for use in threading forms (& elsewhere). Evaluates the supplied
   expression, printing both the expression, its type, and its value to stdout, then returns the value."
  [expr]
  `(let [spy-val#   ~expr
         type-name# (type-name-str spy-val#)]
     (when *spy-enabled*
       (println (str (spy-indent-spaces) '~expr " => <#" type-name# " " (pr-str spy-val#) ">")))
     spy-val#))

(defn ^:no-doc glue-byte-arrays
  "Glues together N byte arrays."
  [& byte-arrays]
  #?(:clj  (let [total-len   (apply + 0 (mapv count byte-arrays))
                 byte-buffer (ByteBuffer/allocate total-len)]
             (doseq [byte-array-curr byte-arrays]
               (.put byte-buffer byte-array-curr))
             (.array byte-buffer))
     :cljs (throw (ex-info "glue-byte-arrays: unimplemented on CLJS" {}))))

(defn glue
  "Glues together like collections:

       (glue [1 2] [3 4] [5 6])                -> [1 2 3 4 5 6]
       (glue {:a 1} {:b 2} {:c 3})             -> {:a 1 :c 3 :b 2}
       (glue #{1 2} #{3 4} #{6 5})             -> #{1 2 6 5 3 4}
       (glue \"I\" \" like \" \\a \" nap!\" )  -> \"I like a nap!\"

  If you want to convert to a sorted set or map, just put an empty one first:

       (glue (sorted-map) {:a 1} {:b 2} {:c 3})      -> {:a 1 :b 2 :c 3}
       (glue (sorted-set) #{1 2} #{3 4} #{6 5})      -> #{1 2 3 4 5 6}

   If there are duplicate keys when using glue for maps or sets, then \"the last one wins\":

       (glue {:band :VanHalen :singer :Dave}  {:singer :Sammy}) "
  [& colls]         ; #todo maybe return nil if no colls?  would allow some-> for users
  (let [string-or-char? #(or (string? %) (char? %))]
    (cond
      (every? #(or (map? %) ; NOTE: this must come first as MapEntry's pass `sequential?`
                   (map-entry? %)) colls) (let [mapentries (drop-if map? colls)
                                                maps       (keep-if map? colls)
                                                me-result  (reduce conj {} mapentries)
                                                result     (reduce into (append maps me-result))]
                                            result)
      (every? sequential? colls) (reduce into [] colls) ; coerce to vector result
      (every? set? colls) (reduce into colls) ; first item determines type of result
      (every? string-or-char? colls) (apply str colls)
      #?(:clj (every? types/byte-array? colls)) #?(:clj (apply glue-byte-arrays colls))

      :else (throw (ex-info "glue: colls must be all same type; found types=" (mapv type colls))))))

(defmacro vals->map ; #todo -> README
  "Called with a list of symbols like `(vals->map a b c)` returns a map
   like {:a a :b b :c c}.

       (let [a 1
             b 2
             c 3]
         (vals->map a b c))  ;=>  {:a 1 :b 2 :c 3} }

   See `with-map-vals` for simple destructuring of such maps."
  [& symbols]
  (let [maps-list (for [symbol symbols]
                    {(keyword symbol) symbol})]
    `(glue ~@maps-list)))

(defmacro with-map-vals ; #todo -> README
  "Given a map like {:a 1 :b 2 :c 3} (such as generated by `(vals->map a b c)`),
  performs safe `let` destructuring using `grab` like:

       (let [some-map  {:a 1 :b 2 :c 3} } ]
         (with-map-vals some-map [a b c]
            (+ a b c)))  ;=>  6

  `with-map-vals` is safe for typos since `grab` will throw if the requested key is not present in the map.
  See `vals->map` for simple creation of labeled data maps."
  [the-map items-vec & forms]
  `(do
     (let           ; generate the binding vector dynamically
       ~(apply glue
               (for [item items-vec
                     :let [sym (symbol (name item))
                           kw  (keyword item)]]
                 [sym (list `grab kw the-map)]))
       ~@forms)))

(defmacro it->
  "A threading macro like as-> that always uses the symbol 'it' as the placeholder for the next threaded value:

        (it-> 1
              (inc it)
              (+ it 3)
              (/ 10 it))
        ;=> 2 "
  [expr & forms]
  `(let [~'it ~expr
         ~@(interleave (repeat 'it) forms)
         ]
     ~'it))

(defn ^:no-doc cond-it-impl
  [expr & forms]
  (let [num-forms (count forms)]
    (assert-info (even? num-forms) "num-forms must be even; value=" (vals->map num-forms forms)))
  (let [cond-action-pairs (partition 2 forms)
        cond-action-forms (for [[cond-form action-form] cond-action-pairs]
                            `(if ~cond-form
                               ~action-form
                               ~'it))]
    `(it-> ~expr ~@cond-action-forms)))

(defmacro cond-it->
  "A threading macro like cond-> that always uses the symbol 'it' as the placeholder for the next threaded value.
  Works in both the conditional form and the value form:

      (let [params {:a 1 :b 1 :c nil :d nil}]
        (cond-it-> params
          (:a it)        (update it :b inc)
          (= (:b it) 2)  (assoc it :c \"here\")
          (:c it)        (assoc it :d \"again\")))

      ;=> {:a 1, :b 2, :c \"here\", :d \"again\"}"
  [& forms]
  (apply cond-it-impl forms))

(defmacro some-it->
  "Threads forms as with `it->`, terminates & returns `nil` if any expression is nil."
  [expr & forms]
  (let [binding-pairs (interleave (repeat 'it) forms)]
    `(let-some [~'it ~expr
                ~@binding-pairs]
               ~'it)))

(defmacro let-some
  "Threads forms as with `when-some`, but allow more than 1 pair of binding forms."
  [bindings & forms]
  (let [num-bindings (count bindings)]
    (assert-info (even? num-bindings) (str "num-bindings must be even; value=" num-bindings) bindings)
    (if (pos? num-bindings)
      `(let [result# ~(cc/second bindings)]
         (when (not-nil? result#)
           (let [~(cc/first bindings) result#]
             (let-some ~(cc/drop 2 bindings) ~@forms))))
      `(do ~@forms))))

(defmacro map-let*
  "Usage:  (map-let* ctx bindings & forms)

  where ctx is a map with default values:
    {:strict true
     :lazy   false}"
  [context bindings & forms]
  (assert-info (not-empty? bindings) "map-let*: bindings cannot be empty=" bindings)
  (assert-info (even? (count bindings)) "map-let*: (count bindings) must be even=" bindings)
  (assert-info (pos? (count forms)) "map-let*: forms cannot be empty=" forms)
  (let [binding-pairs (partition 2 bindings)
        syms          (mapv xfirst binding-pairs)
        colls         (mapv xsecond binding-pairs)]
    `(do
       (assert-info (map? ~context) "map-let*: context must be a map=" ~context)
       (let [lazy#          (get ~context :lazy false)
             strict#        (get ~context :strict true)
             lengths#       (mapv count ~colls)
             lengths-equal# (apply = lengths#)
             map-fn#        (fn ~syms ~@forms)
             output-fn#     (if lazy# identity vec)]
         (when (and strict#
                    (not lengths-equal#))
           (throw (ex-info "map-let*: colls must all be same length" {:lens lengths#})))
         (output-fn# (map map-fn# ~@colls))))))

(defmacro map-let
  "Usage: (map-let bindings & forms)

  Given bindings and forms like

      (map-let [x xs
                y ys]
        (+ x y))

  will iterate over the collections [xs ys] assigning
  successive values of each collection to `x` & `y`, respectively.  Note that the sequences are
  consumed ***in parallel***, and are not nested as with `for` and `doseq`.
  The local symbols `x` & `y` can then be used in `forms` to generate the output mapping.
  Will throw if collections are not all of the same length. Not lazy."
  [bindings & forms]
  `(map-let* {:strict true
              :lazy   false}
             ~bindings ~@forms))

(defn zip*
  "Usage:
        (zip* ctx & colls)
  Like `zip` in Python.  Given 2 or more sequences `colls`, joins the i'th item from each
  collection into a vector, returning a vector of vectors. Context map `ctx` defaults to
  `{:strict true}`. If `:strict` is set, colls must be identical in length.
  Will accept infinite (lazy) sequences like `(range)`. Output is not lazy. "
  [ctx & colls]     ; #todo how use Schema with "rest" args?
  (assert (map? ctx))
  (assert #(every? sequential? colls))
  (let [num-colls  (count colls)
        strict-flg (get ctx :strict true)]
    (loop [result []
           colls  colls]
      (let [empty-flgs  (mapv empty? colls)
            num-empties (count (keep-if truthy? empty-flgs))]
        (if (zero? num-empties)
          (do
            (let [new-row     (mapv xfirst colls)
                  new-results (append result new-row)]
              (recur
                new-results
                (mapv xrest colls))))
          (do
            (when (and strict-flg
                       (not= num-empties num-colls))
              (throw (ex-info "zip*: collections are not all same length; empty-flgs="
                              (vals->map ctx empty-flgs))))
            result))))))

(defn zip
  "Zips together vectors producing a vector of tuples (like Python zip).
  Example:

        (zip
          [:a :b :c]
          [ 1  2  3]
          [:x :y :z)

        ;=>  [ [:a 1 :x]
               [:b 2 :y]
               [:c 3 :z] ]
    Will accept infinite (lazy) sequences like `(range)`. Output is not lazy. "
  [& args]
  (assert #(every? sequential? args))
  (apply zip* {:strict true} args))

(defmacro lazy-cons
  "The simple way to create a lazy sequence:

        (defn lazy-next-int [n]
          (t/lazy-cons n (lazy-next-int (inc n))))

        (def all-ints (lazy-next-int 0))
        "
  [curr-val recursive-call-form]
  `(lazy-seq (cons ~curr-val ~recursive-call-form)))

(s/defn fetch-in :- s/Any
  "A fail-fast version of clojure.core/get-in. When invoked as (fetch-in mappy path),
   returns the value associated with path as for (clojure.core/get-in mappy path).
   Throws an Exception if the path path is not present in mappy."
  [mappy :- (s/cond-pre tsk/Map tsk/Vec)
   path :- tsk/Vec] ; #todo add arity to take default value like `get-in`
  (validate associative? mappy)
  (let [result (get-in mappy path ::not-found)]
    (if (= result ::not-found)
      (throw (ex-info "Key seq not present in map:" {:path path :mappy (snip mappy)}))
      result)))

(s/defn grab :- s/Any
  "A fail-fast version of keyword/map lookup.  When invoked as (grab :the-key the-map),
   returns the value associated with :the-key as for (clojure.core/get the-map :the-key).
   Throws an Exception if :the-key is not present in the-map."
  [the-key :- s/Any
   the-map :- tsk/Map]
  (fetch-in the-map [the-key]))

(defn validate
  "(validate tst-fn tst-val)
  Used to validate intermediate results. Returns tst-val if the result of
  (tst-fn tst-val) is truthy.  Otherwise, throws ex-info with ex-data
  {:sample-val sample-val :tst-result tst-result}."
  [tst-fn tst-val]
  (let [tst-result (tst-fn tst-val)]
    (assert-info (truthy? tst-result) "validate: " (vals->map tst-val tst-result))
    tst-val))

(defmacro verify-form
  "
         (verify <some-expr>)

  Used to verify intermediate results. Returns value of `<some-expr>` if the result
  is truthy.  Otherwise, throws an Exception."
  [form]
  `(let [value# ~form]
     (if (truthy? value#)
       value#
       (throw (ex-info "verification failed  " '~form)))))

(s/defn idx
  "Indexes into a vector, allowing negative index values"
  [coll :- tsk/List
   index-val :- s/Int]
  (assert-info (not-nil? coll) "idx: coll cannot be nil: " (vals->map coll))
  (let [data-vec (vec coll)
        N        (count data-vec)
        >>       (assert (pos? N))
        ii       (mod index-val N)
        >>       (assert-info (< ii (count coll)) "Index cannot exceed collection length: " {:len (count coll) :index ii})
        result   (clojure.core/get data-vec ii)]
    result))

(s/defn sequential->idx-map :- {s/Any s/Any} ; #todo move
  [data :- [s/Any]]
  (into (sorted-map)
        (map-indexed (fn [idx val] [idx val])
                     data)))

(defn char->sym [ch] (symbol (str ch)))

(defn get-in-strict [data path] ; #todo move
  (let [result (get-in data path ::not-found)]
    (when (= result ::not-found)
      (throw (ex-info "destruct(get-in-strict): value not found" {:data data :path path})))
    result))

(defn ^:no-doc destruct-tmpl-analyze
  [ctx]
  (with-map-vals ctx [parsed path tmpl]
    ;(spyx path)
    (cond
      (map? tmpl)
      (doseq [entry tmpl]
        ;(spyx entry)
        (let [[curr-key curr-val] entry]
          ;(spyx [curr-key curr-val])
          (let [path-new (append path curr-key)]
            ;(spyx path-new)
            (if (symbol? curr-val)
              (let [var-sym (if (= curr-val (char->sym \?))
                              (kw->sym curr-key)
                              curr-val)]
                (swap! parsed append {:path path-new :name var-sym}))
              (destruct-tmpl-analyze {:parsed parsed :path path-new :tmpl curr-val})))))

      (sequential? tmpl)
      (do
        ;(spy :tmpl-51 tmpl)
        (destruct-tmpl-analyze {:parsed parsed :path path :tmpl (sequential->idx-map tmpl)}))

      :else (println :oops-44))))

(defn ^:no-doc is-restruct-one?
  "Return true if receive a form like either `(restruct)` or `(restruct info)` (i.e. either zero or one symbol args)."
  [form]
  (and (list? form)
       (= 2 (count form))
       (= 'restruct (cc/first form))))

(defn ^:no-doc destruct-impl
  [bindings forms]
  (assert-info (even? (count bindings)) "destruct: uneven number of bindings:" bindings)
  (assert-info (not-empty? bindings) "destruct: bindings empty:" bindings)
  (let [binding-pairs (partition 2 bindings)
        datas         (mapv cc/first binding-pairs)
        tmpls         (mapv cc/second binding-pairs)
        tmpls-parsed  (vec (for [tmpl tmpls]
                             (let [parsed (atom [])]
                               (destruct-tmpl-analyze {:parsed parsed :path [] :tmpl tmpl})
                               @parsed)))]
    ; (spyx tmpls-parsed)
    ; look for duplicate variable names
    (let [var-names (vec (for [tmpl-parsed   tmpls-parsed
                               path-name-map tmpl-parsed]
                           (grab :name path-name-map)))]
      ; (spyx var-names)
      (when (not= var-names (distinct var-names))
        (println "destruct: var-names not unique" var-names)
        (throw (ex-info "destruct: var-names not unique" var-names))))

    (let [data-parsed-pairs (zip datas tmpls-parsed)]
      ; (spyx data-parsed-pairs)
      (let [extraction-pairs    (apply glue
                                       (for [[data parsed] data-parsed-pairs
                                             {:keys [name path]} parsed]
                                         [name `(get-in-strict ~data ~path)]))
            ; >>   (do (nl) (spyx extraction-pairs))

            construct-one-pairs (apply glue
                                       (for [[data parsed] data-parsed-pairs]
                                         {data (apply glue
                                                      (for [{:keys [name path]} parsed]
                                                        `[~data (assoc-in ~data ~path ~name)]))}))
            ; >>   (do (nl) (spyx-pretty construct-one-pairs))

            construct-all-pairs (apply glue (vals construct-one-pairs))
            ; >>   (spyx-pretty construct-all-pairs)

            restruct-one-defs   (apply glue
                                       (for [[data construction-pairs] construct-one-pairs]
                                         {data (apply list `[fn []
                                                             (let [~@construction-pairs]
                                                               ~data)])}))
            ; >>   (do (nl) (spyx-pretty restruct-one-defs))

            restruct-only-def   (when (= 1 (count datas))
                                  (let [[data construction-pairs] (only construct-one-pairs)] ; #todo test single data case
                                    (apply list `[fn []
                                                  (let [~@construction-pairs]
                                                    ~data)])))
            ; >>   (do (nl) (spyx-pretty restruct-only-def))

            restruct-all-def    (apply list `[fn []
                                              (let [~@construct-all-pairs]
                                                (vals->map ~@datas))])
            ; >>   (do (nl) (spyx-pretty restruct-all-def))

            res-raw             `(let [~@extraction-pairs]
                                   ~@forms)
            ; >>   (do (nl) (spyx-pretty res-raw))

            ; #todo fix to work for "namespaced" calls like (t/restruct) & (t/restruct-all)
            res-all             (walk/postwalk
                                  (fn [form]
                                    (if (not= form '(restruct-all))
                                      form
                                      (list 'let ['restruct-fn restruct-all-def
                                                  'result (list 'restruct-fn)]
                                            'result)))
                                  res-raw)
            ; >>   (do (nl) (spyx-pretty res-all))

            res-one             (walk/postwalk
                                  (fn [form]
                                    (if-not (is-restruct-one? form)
                                      form
                                      (let [restr-one-data (cc/second form)
                                            restr-one-def  (get restruct-one-defs restr-one-data ::not-found)]
                                        (list 'let ['restruct-fn restr-one-def
                                                    'result (list 'restruct-fn)]
                                              'result))))
                                  res-all)
            ; >>   (do (nl) (spyx-pretty res-one))

            res-only            (walk/postwalk
                                  (fn [form]
                                    (if (not= form '(restruct))
                                      form
                                      (if (not= 1 (count datas)) ; #todo test exception works
                                        (do
                                          (println "(restruct) error: more than 1 data src present" datas)
                                          (throw (ex-info "restruct:  aborting..." (:count (count datas)))))
                                        (list 'let ['restruct-fn restruct-only-def
                                                    'result (list 'restruct-fn)]
                                              'result))))
                                  res-one)]
        ; (do (nl) (spyx-pretty res-only))
        res-only))))

(defmacro destruct
  "Natural destructuring:

       (let [data {:a 1
                   :b {:c 3
                       :d 4}}]
         ...
         (destruct [data {:a ?
                          :b {:c ?}}]
         ...

   then can use local values  a=1, c=3.  With vector data:

       (let [data [1 2 3 4 5]]
         ...
         (destruct [data [a b c]]
          ...

   then can use local values a=1 b=2 c=3.  Can use `(restruct)`, `(restruct data)`, or `(restruct-all)`
   to re-structure & return original data shape using current values."
  [bindings & forms]
  (destruct-impl bindings forms))

(defn restruct
  "within a `(destruct [<data> <shape>] ...)` form, `(restruct)` or `(restruct <data>)`
   causes re-structuring & return of original data shape using current values."
  [& args] (throw (ex-info "restruct: illegal usage - should never get here." args)))

(defn restruct-all
  "Within a form

      (destruct [data-1 <shape-1>
                 data-2 <shape-2] ...)

   causes re-structuring & return of original data shapes using
   current values as with

        (vals->map data-1 data-2 ...)
  "
  [& args] (throw (ex-info "restruct-all: illegal usage - should never get here." args)))

(defn pretty
  "Shortcut for (clojure.pprint/pprint (unlazy-pretty data))."
  [data] (pprint/pprint (unlazy-pretty data)))

(defn pretty-str
  "Returns a string that is the result of tupelo.core/pretty"
  [data] (with-out-str (pretty data)))
