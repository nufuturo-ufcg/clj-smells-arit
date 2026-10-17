(ns cljscm.core
  (:require [cljscm.conditional :as condc]))

(defmacro and
  "Evaluates exprs one at a time, from left to right. If a form
  returns logical false (nil or false), and returns that value and
  doesn't evaluate any of the other expressions, otherwise it returns
  the value of the last expr. (and) returns true."
  {:added "1.0"}
  ([] true)
  ([x] x)
  ([x & next]
   `(let [and# ~x]
      (if and# (and ~@next) and#))))

(defmacro assert
  "Evaluates expr and throws an exception if it does not evaluate to
  logical true."
  {:added "1.0"}
  ([x]
     (when *assert*
       `(when-not ~x
          (throw (new AssertionError (str "Assert failed: " (pr-str '~x)))))))
  ([x message]
     (when *assert*
       `(when-not ~x
          (throw (new AssertionError (str "Assert failed: " ~message "\n" (pr-str '~x))))))))

(defmacro cond
  "Takes a set of test/expr pairs. It evaluates each test one at a
  time.  If a test returns logical true, cond evaluates and returns
  the value of the corresponding expr and doesn't evaluate any of the
  other tests or exprs. (cond) returns nil."
  {:added "1.0"}
  [& clauses]
    (when clauses
      (list 'if (first clauses)
            (if (next clauses)
                (second clauses)
                (throw (IllegalArgumentException.
                         "cond requires an even number of forms")))
            (cons 'clojure.core/cond (next (next clauses))))))

(def ^{:private true :dynamic true}
  assert-valid-fdecl (fn [fdecl]))

(defmacro defn-
  "same as defn, yielding non-public def"
  {:added "1.0"}
  [name & decls]
  (list* `defn (with-meta name (assoc (meta name) :private true)) decls))

(defmacro ^{:private true} assert-args [fnname & pairs]
  `(do (when-not ~(first pairs)
         (throw (IllegalArgumentException.
                  ~(core/str fnname " requires " (second pairs)))))
     ~(core/let [more (nnext pairs)]
        (when more
          (list* `assert-args fnname more)))))

(defmacro for
  "List comprehension. Takes a vector of one or more
   binding-form/collection-expr pairs, each followed by zero or more
   modifiers, and yields a lazy sequence of evaluations of expr.
   Collections are iterated in a nested fashion, rightmost fastest,
   and nested coll-exprs can refer to bindings created in prior
   binding-forms.  Supported modifiers are: :let [binding-form expr ...],
   :while test, :when test.

  (take 100 (for [x (range 100000000) y (range 1000000) :while (< y x)] [x y]))"
  {:added "1.0"}
  [seq-exprs body-expr]
  (assert-args
     (vector? seq-exprs) "a vector for its binding"
     (even? (count seq-exprs)) "an even number of forms in binding vector")
  (let [to-groups (core/fn [seq-exprs]
                    (reduce1 (core/fn [groups [k v]]
                              (if (keyword? k)
                                (conj (pop groups) (conj (peek groups) [k v]))
                                (conj groups [k v])))
                            [] (partition 2 seq-exprs)))
        err (core/fn [& msg] (throw (IllegalArgumentException. ^String (apply str msg))))
        emit-bind (core/fn emit-bind [[[bind expr & mod-pairs]
                                  & [[_ next-expr] :as next-groups]]]
                    (let [giter (gensym "iter__")
                          gxs (gensym "s__")
                          do-mod (core/fn do-mod [[[k v :as pair] & etc]]
                                   (cond
                                     (= k :let) `(let ~v ~(do-mod etc))
                                     (= k :while) `(when ~v ~(do-mod etc))
                                     (= k :when) `(if ~v
                                                    ~(do-mod etc)
                                                    (recur (rest ~gxs)))
                                     (keyword? k) (err "Invalid 'for' keyword " k)
                                     next-groups
                                      `(let [iterys# ~(emit-bind next-groups)
                                             fs# (seq (iterys# ~next-expr))]
                                         (if fs#
                                           (concat fs# (~giter (rest ~gxs)))
                                           (recur (rest ~gxs))))
                                     :else `(cons ~body-expr
                                                  (~giter (rest ~gxs)))))]
                      (if next-groups
                        #_"not the inner-most loop"
                        `(fn ~giter [~gxs]
                           (lazy-seq
                             (loop [~gxs ~gxs]
                               (when-first [~bind ~gxs]
                                 ~(do-mod mod-pairs)))))
                        #_"inner-most loop"
                        (let [gi (gensym "i__")
                              gb (gensym "b__")
                              do-cmod (core/fn do-cmod [[[k v :as pair] & etc]]
                                        (cond
                                          (= k :let) `(let ~v ~(do-cmod etc))
                                          (= k :while) `(when ~v ~(do-cmod etc))
                                          (= k :when) `(if ~v
                                                         ~(do-cmod etc)
                                                         (recur
                                                           (unchecked-inc ~gi)))
                                          (keyword? k)
                                            (err "Invalid 'for' keyword " k)
                                          :else
                                            `(do (chunk-append ~gb ~body-expr)
                                                 (recur (unchecked-inc ~gi)))))]
                          `(fn ~giter [~gxs]
                             (lazy-seq
                               (loop [~gxs ~gxs]
                                 (when-let [~gxs (seq ~gxs)]
                                   (if (chunked-seq? ~gxs)
                                     (let [c# (chunk-first ~gxs)
                                           size# (int (count c#))
                                           ~gb (chunk-buffer size#)]
                                       (if (loop [~gi (int 0)]
                                             (if (< ~gi size#)
                                               (let [~bind (.nth c# ~gi)]
                                                 ~(do-cmod mod-pairs))
                                               true))
                                         (chunk-cons
                                           (chunk ~gb)
                                           (~giter (chunk-rest ~gxs)))
                                         (chunk-cons (chunk ~gb) nil)))
                                     (let [~bind (first ~gxs)]
                                       ~(do-mod mod-pairs)))))))))))]
    `(let [iter# ~(emit-bind (to-groups seq-exprs))]
       (iter# ~(second seq-exprs)))))

(defmacro or
  "Evaluates exprs one at a time, from left to right. If a form
  returns a logical true value, or returns that value and doesn't
  evaluate any of the other expressions, otherwise it returns the
  value of the last expression. (or) returns nil."
  {:added "1.0"}
  ([] nil)
  ([x] x)
  ([x & next]
      `(let [or# ~x]
         (if or# or# (or ~@next)))))

(defmacro when
  "Evaluates test. If logical true, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (list 'if test (cons 'do body)))

(defmacro when-first
  "bindings => x xs

  Roughly the same as (when (seq xs) (let [x (first xs)] body)) but xs is evaluated only once"
  {:added "1.0"}
  [bindings & body]
  (assert-args
     (vector? bindings) "a vector for its binding"
     (= 2 (count bindings)) "exactly 2 forms in binding vector")
  (let [[x xs] bindings]
    `(when-let [xs# (seq ~xs)]
       (let [~x (first xs#)]
         ~@body))))

(defmacro when-let
  "bindings => binding-form test

  When test is true, evaluates body with binding-form bound to the value of test"
  {:added "1.0"}
  [bindings & body]
  (assert-args
     (vector? bindings) "a vector for its binding"
     (= 2 (count bindings)) "exactly 2 forms in binding vector")
   (let [form (bindings 0) tst (bindings 1)]
    `(let [temp# ~tst]
       (when temp#
         (let [~form temp#]
           ~@body)))))

(defmacro when-not
  "Evaluates test. If logical false, evaluates body in an implicit do."
  {:added "1.0"}
  [test & body]
  (list 'if test nil (cons 'do body)))

(defmacro while
  "Repeatedly executes body while test expression is true. Presumes
  some side-effect will cause test to become false/nil. Returns nil"
  {:added "1.0"}
  [test & body]
  `(loop []
     (when ~test
       ~@body
       (recur))))

(def compare-arglist
  (comparator
   (fn* [arglist-a arglist-b]
     (if (some #{'&} arglist-a)
       false
       (if (some #{'&} arglist-b)
         true
         (core/< (count arglist-a) (count arglist-b)))))))

(defmacro scm* [& forms] (case condc/*target-platform* :gambit (cons 'scm* forms)))

(core/defn
  maybe-destructured
  [params body]
  (if (every? symbol? params)
    (cons params body)
    (core/loop [params params
           new-params []
           lets []]
      (if params
        (if (symbol? (first params))
          (recur (next params) (conj new-params (first params)) lets)
          (core/let [gparam (gensym "p__")]
            (recur (next params) (conj new-params gparam)
                   (-> lets (conj (first params)) (conj gparam)))))
        `(~new-params
          (let ~lets
            ~@body))))))

(core/defn destructure [bindings]
  (core/let [bents (partition 2 bindings)
         pb (core/fn pb [bvec b v]
              (core/let [pvec
                     (core/fn [bvec b val]
                       (core/let [gvec (gensym "vec__")]
                         (core/loop [ret (-> bvec (conj gvec) (conj val))
                                     n 0
                                     bs b
                                     seen-rest? false]
                           (if (core/seq bs)
                             (core/let [firstb (first bs)]
                               (cond
                                 (= firstb '&) (recur (pb ret (second bs) (list `nthnext gvec n))
                                                      n
                                                      (nnext bs)
                                                      true)
                                 (= firstb :as) (pb ret (second bs) gvec)
                                 :else (if seen-rest?
                                         (throw (new Exception "Unsupported binding form, only :as can follow & parameter"))
                                         (recur (pb ret firstb  (list `nth gvec n nil))
                                                (core/inc n)
                                                (next bs)
                                                seen-rest?))))
                             ret))))
                     pmap
                     (core/fn [bvec b v]
                       (core/let [gmap (gensym "map__")
                                  defaults (:or b)]
                         (core/loop [ret (-> bvec (conj gmap) (conj v)
                                             (conj gmap) (conj `(if (core/seq? ~gmap) (apply hash-map ~gmap) ~gmap))
                                             ((core/fn [ret]
                                                (if (:as b)
                                                  (conj ret (:as b) gmap)
                                                  ret))))
                                     bes (reduce
                                          (core/fn [bes entry]
                                            (reduce #(assoc %1 %2 ((val entry) %2))
                                                    (dissoc bes (key entry))
                                                    ((key entry) bes)))
                                          (dissoc b :as :or)
                                          {:keys #(keyword (core/str %)), :strs core/str, :syms #(list `quote %)})]
                           (if (core/seq bes)
                             (core/let [bb (key (first bes))
                                        bk (val (first bes))
                                        has-default (contains? defaults bb)]
                               (recur (pb ret bb (if has-default
                                                   (list `get gmap bk (defaults bb))
                                                   (list `get gmap bk)))
                                      (next bes)))
                             ret))))]
                    (cond
                      (symbol? b) (-> bvec (conj b) (conj v))
                      (vector? b) (pvec bvec b v)
                      (map? b) (pmap bvec b v)
                      :else (throw (new Exception (core/str "Unsupported binding form: " b))))))
         process-entry (core/fn [bvec b] (pb bvec (first b) (second b)))]
        (if (every? symbol? (map first bents))
          bindings
          (reduce process-entry [] bents))))

(defmacro let
  "binding => binding-form init-expr

  Evaluates the exprs in a lexical context in which the symbols in
  the binding-forms are bound to their respective init-exprs or parts
  therein."
  [bindings & body]
  (assert-args
     (vector? bindings) "a vector for its binding"
     (even? (count bindings)) "an even number of forms in binding vector")
  `(let* ~(destructure bindings) ~@body))

(defmacro loop
  "Evaluates the exprs in a lexical context in which the symbols in
  the binding-forms are bound to their respective init-exprs or parts
  therein. Acts as a recur target."
  [bindings & body]
    (assert-args
      (vector? bindings) "a vector for its binding"
      (even? (count bindings)) "an even number of forms in binding vector")
    (core/let [db (destructure bindings)]
      (if (= db bindings)
        `(loop* ~bindings ~@body)
        (core/let [vs (take-nth 2 (drop 1 bindings))
              bs (take-nth 2 bindings)
              gs (map (core/fn [b] (if (symbol? b) b (gensym))) bs)
              bfs (reduce (core/fn [ret [b v g]]
                            (if (symbol? b)
                              (conj ret g v)
                              (conj ret g v b g)))
                          [] (map vector bs vs gs))]
          `(let ~bfs
             (loop* ~(vec (interleave gs gs))
               (let ~(vec (interleave bs gs))
                 ~@body)))))))

(defmacro scm-length [lst]
  `(scm* {:lst ~lst} ~'(length :lst)))

(defmacro scm-nillify [lst]
  `(scm* {:lst ~lst :nil ~nil} ~'(if (null? :lst) :nil :lst)))

(core/defn make-car [lst cnt]
  (if (core/<= cnt 3) (core/list (core/symbol (core/str "ca" (apply str (repeat cnt "d")) "r")) lst)
      (core/list 'list-ref lst cnt)))

(core/defn make-cdr [lst cnt]
  (if (core/= cnt 0) lst
      (if (core/<= cnt 4) (core/list (core/symbol (core/str "c" (apply str (repeat cnt "d")) "r")) lst)
          (make-cdr (make-cdr lst 4) (core/- cnt 4)))))

(core/defn- combine-arities
  [methods]
  (let [[smallest-meth & other-meths :as meths] (sort-by first compare-arglist methods)
        middle-meths (butlast other-meths)
        biggest-meth (last other-meths)
        num-shared (count (first smallest-meth))
        restparam (gensym "rest")
        find-inits
        , (core/fn [meth]
            (let [args (first meth)
                  variadic? (some #{'&} args)
                  shared-args (take num-shared args)
                  fixed-args (if variadic?
                               (->> args (drop num-shared) (drop-last 2))
                               (drop num-shared args))
                  variadic-arg (when variadic? (last args))
                  inits (concat
                         (map vector shared-args (first smallest-meth))
                         (map-indexed (core/fn [i a] [a `(~'scm* [~restparam] ~(make-car restparam i))])
                                      fixed-args)
                         (when variadic? [[variadic-arg
                                             `(scm-nillify (~'scm* [~restparam] ~(make-cdr restparam (core/count fixed-args))))]]))]
              {:args (map (comp #(if (= '_ %) (gensym "_") %) first) inits) ; munge possibly colliding _'s
               :inits (map second inits)
               :implforms (rest meth) 
               :restcount (when (not variadic-arg) (core/count fixed-args))}))]
    {:meths (map find-inits meths)
     :restparam restparam
     :num-shared num-shared 
     :combined-argform (vec (concat (first smallest-meth) ['& restparam]))}))

(defmacro fn
  "params => positional-params* , or positional-params* & next-param
  positional-param => binding-form
  next-param => binding-form
  name => symbol

  Defines a function"
  {:added "1.0", :special-form true,
   :forms '[(fn name? [params* ] exprs*) (fn name? ([params* ] exprs*)+)]}
  [& sigs]
  (core/let [name (if (symbol? (first sigs)) (first sigs) nil)
             sigs (if name (next sigs) sigs)
             sigs (if (vector? (first sigs)) (list sigs) sigs)
             psig (fn* [sig]
                       (core/let [[params & body] sig
                                  conds (when (and (next body) (map? (first body))) 
                                          (first body))
                                  body (if conds (next body) body)
                                  conds (or conds (meta params))
                                  pre (:pre conds)
                                  post (:post conds)                       
                                  body (if post
                                         `((let [~'% ~(if (core/< 1 (count body)) 
                                                        `(do ~@body) 
                                                        (first body))]
                                             ~@(map (fn* [c] `(assert ~c)) post)
                                             ~'%))
                                         body)
                                  body (if pre
                                         (concat (map (fn* [c] `(assert ~c)) pre) 
                                                 body)
                                         body)]
                         (maybe-destructured params body)))
             dsigs (map psig sigs)
             any-variadic? (core/some #(some #{'&} (first %)) dsigs)
             single-arity? (core/<= (count dsigs) 1)
             combined-form
             , (if single-arity?
                 dsigs
                 (let [{:keys [meths combined-argform restparam num-shared]}
                       , (combine-arities dsigs)
                       meths (if any-variadic?
                               meths
                               (concat meths [{:implforms `(throw (cljscm.core/Error. (str "No arity method for " (+ ~num-shared (scm-length ~restparam)) " args.")))}]))]
                   `(~combined-argform
                     (~'case (scm-length ~restparam)
                       ~@(mapcat (core/fn [{:keys [args inits implforms restcount]}]
                                   (let [initpairs
                                         , (->> (map vector args inits)
                                                ;(filter (core/fn [[a b]] (not= a b))) ; no re-binding needed if method arg is same as the combined arg. (but we need a potential recur target, so we need to bind everything for now.)
                                                (apply concat)
                                                (core/seq)) 
                                         body (if true ;initpairs
                                                `(~'loop [~@initpairs]
                                                   ~@implforms)
                                                `(do ~@implforms))]
                                     (if restcount
                                       [restcount body]
                                       [body])))
                                 meths)))))]
    (with-meta
      (if name
        (list* 'fn* name combined-form)
        (cons 'fn* combined-form))
      (assoc (meta &form)
        :single-arity single-arity?))))

(defmacro lazy-seq [& body]
  `(new cljscm.core/LazySeq nil false (core/fn [] ~@body) nil))

(defmacro binding
  "binding => var-symbol init-expr

  Creates new bindings for the (already-existing) vars, with the
  supplied initial values, executes the exprs in an implicit do, then
  re-establishes the bindings that existed before.  The new bindings
  are made in parallel (unlike let); all init-exprs are evaluated
  before the vars are bound to their new values."
  [bindings & body]
  (core/let [names (take-nth 2 bindings)
             qualnames (map #(:name (ana/resolve-var (dissoc &env :locals) %)) names)
             vals (take-nth 2 (drop 1 bindings))
             tempnames (map (comp gensym name) names)
             binds (map vector names vals)
             resets (reverse (map vector names tempnames))]
    (ana/confirm-bindings &env names)
    `(~'(scm* {} parameterize)
      (~'scm* ~(apply hash-map (interleave tempnames vals)) ~(map #(list %1 %2) qualnames tempnames))
      ~@body)))

(defmacro assert
  "Evaluates expr and throws an exception if it does not evaluate to
  logical true."
  ([x]
     (when *assert*
       `(when-not ~x
          (throw (cljscm.core/Error.
                  (cljscm.core/str "Assert failed: " (cljscm.core/pr-str '~x)))))))
  ([x message]
     (when *assert*
       `(when-not ~x
          (throw (cljscm.core/Error.
                  (cljscm.core/str "Assert failed: " ~message "\n" (cljscm.core/pr-str '~x))))))))

(defmacro for
  "List comprehension. Takes a vector of one or more
   binding-form/collection-expr pairs, each followed by zero or more
   modifiers, and yields a lazy sequence of evaluations of expr.
   Collections are iterated in a nested fashion, rightmost fastest,
   and nested coll-exprs can refer to bindings created in prior
   binding-forms.  Supported modifiers are: :let [binding-form expr ...],
   :while test, :when test.

  (take 100 (for [x (range 100000000) y (range 1000000) :while (< y x)]  [x y]))"
  [seq-exprs body-expr]
  (assert-args for
     (vector? seq-exprs) "a vector for its binding"
     (even? (count seq-exprs)) "an even number of forms in binding vector")
  (core/let [to-groups (core/fn [seq-exprs]
                    (reduce (core/fn [groups [k v]]
                              (if (keyword? k)
                                (conj (pop groups) (conj (peek groups) [k v]))
                                (conj groups [k v])))
                            [] (partition 2 seq-exprs)))
        err (core/fn [& msg] (throw (apply str msg)))
        emit-bind (core/fn emit-bind [[[bind expr & mod-pairs]
                                  & [[_ next-expr] :as next-groups]]]
                    (core/let [giter (gensym "iter__")
                          gxs (gensym "s__")
                          do-mod (core/fn do-mod [[[k v :as pair] & etc]]
                                   (cond
                                     (= k :let) `(let ~v ~(do-mod etc))
                                     (= k :while) `(when ~v ~(do-mod etc))
                                     (= k :when) `(if ~v
                                                    ~(do-mod etc)
                                                    (recur (rest ~gxs)))
                                     (keyword? k) (err "Invalid 'for' keyword " k)
                                     next-groups
                                      `(let [iterys# ~(emit-bind next-groups)
                                             fs# (core/seq (iterys# ~next-expr))]
                                         (if fs#
                                           (concat fs# (~giter (rest ~gxs)))
                                           (recur (rest ~gxs))))
                                     :else `(cons ~body-expr
                                                  (~giter (rest ~gxs)))))]
                      `(fn ~giter [~gxs]
                         (lazy-seq
                           (loop [~gxs ~gxs]
                             (when-first [~bind ~gxs]
                               ~(do-mod mod-pairs)))))))]
    `(let [iter# ~(emit-bind (to-groups seq-exprs))]
       (iter# ~(second seq-exprs)))))

(defmacro aclone-push
  "Clones the array into an array one element larger; with the value pushed."
  [a o]
   `(let [sz# (scm* {:a ~a} ~'(vector-length :a))
         r# (scm* {:nsz (inc sz#)} ~'(make-vector :nsz))]
      (scm* {:a ~a :sz sz# :r r# :o ~o}
            ~'(begin
               (subvector-move! :a 0 :sz :r 0)
               (vector-set! :r :sz :o)))
      r#))

(defmacro time
  "Evaluates expr and prints the time it took. Returns the value of expr."
  [expr]
  `(let [start# (.getTime (js/Date.))
         ret# ~expr]
     (prn (core/str "Elapsed time: " (- (.getTime (js/Date.)) start#) " msecs"))
     ret#))
