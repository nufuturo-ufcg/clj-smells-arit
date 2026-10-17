(ns emmy.function
  "Procedures that act on Clojure's function and multimethod types, along with
  extensions of the Emmy generic operations to functions.

  See [the `Function`
  cljdocs](https://cljdoc.org/d/org.mentat/emmy/CURRENT/doc/data-types/function)
  for a discussion of generic function arithmetic."
  (:refer-clojure :exclude [get get-in memoize with-meta name])
  (:require [clojure.core :as core]
            [clojure.core.match :refer [match]]
            [emmy.dual :as d]
            [emmy.generic :as g]
            [emmy.util :as u]
            [emmy.value :as v])
  #?(:clj
     (:import (clojure.lang AFunction Fn RestFn MultiFn Keyword Symbol Var)
              (java.lang.reflect Method))))

(defn- with-meta
  "The current Clojurescript definition of `with-meta` first tests its
   argument with `js-fn?` and generates a MetaFn if so, frustrating our
   definition of IWithMeta on native JS function objects. This wrapper
   delegates to our definition, which allows native functions to safely
   carry metadata. Note that in Clojure one is guaranteed a fresh object
   with the new metadata, but in Clojurescript the target is mutated.
   This function is safe to use on freshly created functions, but may
   require careful consideration in other contexts."
  [f m]
  #?(:cljs (v/set-js-meta! f m)
     :clj (clojure.core/with-meta f m)))

(defn with-arity
  "Appends the supplied `arity` to the metadata of `f`, knocking out any
  pre-existing arity notation.

  Optionally accepts a third parameter `m` of metadata to attach to the return
  function, in addition to the new `:arity` key."
  ([f arity]
   (with-arity f arity {}))
  ([f arity m]
   (let [new-meta (-> (meta f)
                      (merge m)
                      (assoc :arity arity))]
     (with-meta f new-meta))))

(def ^:dynamic *strict-arity-checks*
  "If true, attempting to pass two functions of incompatible arity
  into any binary function, or into [[combine-arities]], will throw. False by
  default."
  false)

(defn combine-arities
  "Returns the joint arity of arities `a` and `b`.

  The joint arity is the loosest possible arity specification compatible with
  both `a` and `b`. Throws if `a` and `b` are incompatible."
  ([] [:at-least 0])
  ([a] a)
  ([a b]
   (letfn [(fail []
             (if *strict-arity-checks*
               (u/illegal (str "Incompatible arities: " a " " b))
               [:at-least 0]))]
     ;; since the combination operation is symmetric, sort the arguments
     ;; so that we only have to implement the upper triangle of the
     ;; relation.
     (if (pos? (compare (first a) (first b)))
       (combine-arities b a)
       (match [a b]
         [[:at-least k] [:at-least k2]] [:at-least (max k k2)]
         [[:at-least k] [:between m n]] (let [m (max k m)]
                                          (cond (= m n) [:exactly m]
                                                (< m n) [:between m n]
                                                :else (fail)))
         [[:at-least k] [:exactly l]] (if (>= l k)
                                        [:exactly l]
                                        (fail))
         [[:between m n] [:between m2 n2]] (let [m (max m m2)
                                                 n (min n n2)]
                                             (cond (= m n) [:exactly m]
                                                   (< m n) [:between m n]
                                                   :else (fail)))
         [[:between m n] [:exactly k]] (if (<= m k n)
                                         [:exactly k]
                                         (fail))
         [[:exactly k] [:exactly l]] (if (= k l) [:exactly k] (fail)))))))

(defn joint-arity
  "Find the most relaxed possible statement of the joint arity of the given sequence of `arities`.
  If they are incompatible, an exception is thrown."
  [arities]
  (reduce combine-arities arities))

(defn coerce-to-fn
  "Given an [[emmy.value/scalar?]] input `x`, returns a function of arity `arity`
  that always returns `x` no matter what input it receives.

  For non-numerical `x`, returns `x`."
  ([x arity]
   (if (v/scalar? x)
     (-> (constantly x)
         (with-arity arity))
     x)))

(defn- binary-operation
  "Accepts a binary function `op`, and returns a function of two functions `f` and
  `g` which will produce the pointwise operation `op` of the results of applying
  both `f` and `g` to the input.

  For example:

  ```clojure
  (([[binary-operation]] op) f g)
  ;;=> (fn [x] (op (f x) (g x)))
  ```"
  [op]
  (letfn [(h [f g]
            (let [f-arity (if (v/scalar? f) (arity g) (arity f))
                  g-arity (if (v/scalar? g) f-arity   (arity g))
                  f1      (coerce-to-fn f f-arity)
                  g1      (coerce-to-fn g g-arity)
                  arity   (joint-arity [f-arity g-arity])]
              (with-arity (fn [& args] (op (apply f1 args) (apply g1 args))) arity)))]
    (with-arity h [:exactly 2])))

(defn- defbinary
  "Given a generic binary function `generic-op` (and an optional `binary-op` to
  perform the work), define the multimethods necessary to introduce this
  operation to function arguments."
  ([generic-op] (defbinary generic-op generic-op))
  ([generic-op binary-op]
   (let [binop (binary-operation binary-op)]
     (doseq [signature [[::v/function ::v/function]
                        [::v/function ::cofunction]
                        [::cofunction ::v/function]]]
       (defmethod generic-op signature [a b]
         (binop a b))))))
