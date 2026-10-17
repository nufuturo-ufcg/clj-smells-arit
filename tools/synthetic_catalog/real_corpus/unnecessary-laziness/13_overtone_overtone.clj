(ns
    ^{:doc "CGens, or Composite Generators are composite UGens and/or other CGens. These are purely an Overtone abstraction whereas UGens have matching counterparts on SuperCollider Server. CGens allow you to build and share re-usable sub-synth components which act like ugens."
      :author "Sam Aaron"}
    overtone.sc.defcgen

  (:use [clojure.walk :as walk]
        [overtone.helpers lib]
        [overtone.sc ugens]
        [overtone.sc.machinery.ugen defaults fn-gen doc]))

(defn- syms->sym-gensym-pairs
  "Given a seq of symbols, creates a new seq of symbol gensym pairs"
  [syms]
  (into {} (doall (map (fn [sym] [(symbol (name sym)) (gensym (name sym))]) syms))))

(defn- mk-cgen-fn
  "Make the function which gets executed when a cgen is called."
  [params body]
  (let [expand-flags (map #(or (:expands? %) false) params)
        param-names  (vec (map :name params))
        defaults     (reduce (fn [s el] (assoc s (:name el) (:default el)))
                             {}
                             params)
        arg-sym      (gensym 'arg-sym)
        sym-gensyms  (syms->sym-gensym-pairs param-names)
        bindings     (reduce (fn [final param]
                               (conj final (get sym-gensyms (symbol (name param))) `(get (arg-mapper ~arg-sym ~param-names ~defaults) ~param)))
                             []
                             param-names)
        body         (walk/prewalk-replace sym-gensyms body )
        cgen-fn      `(fn [& ~arg-sym]
                        (let [~@bindings]
                          (with-overloaded-ugens
                            ~body)))]

    (if (or (empty? params)
            (not-any? true? expand-flags))
      cgen-fn
      `(make-expanding ~cgen-fn (quote ~expand-flags)))))

(defn generate-full-cgen-doc
  "Generate a full docstring from a the specified cgen information"
  ([c-name summary doc categories rate params rates] (generate-full-cgen-doc c-name summary doc categories rate params rates nil nil))
  ([c-name summary doc categories rate params rates src-str contributor]
     (let [spec {:name         c-name
                 :summary      summary
                 :doc          doc
                 :categories   categories
                 :default-rate rate
                 :args         (map stringify-map-vals params)
                 :rates        (into #{} rates)
                 :src-str      src-str
                 :contributor  contributor}]
       (:full-doc (with-full-doc spec)))))

(defn mk-cgen
  "Generate the form representing a cgen - a callable map of associated information
  and the function that evaluates the body within the binding context of the
  params."
  ([c-name summary doc params body categories rate] (mk-cgen c-name summary doc params body categories rate #{rate}))
  ([c-name summary doc params body categories rate rates]
     (let [full-doc (generate-full-cgen-doc c-name summary doc categories rate params rates)
           cgen-fn  (mk-cgen-fn params body)]
       `(callable-map {:params ~params
                       :summary ~summary
                       :doc ~doc
                       :full-doc ~full-doc
                       :categories ~categories
                       :rate ~rate
                       :name ~(name c-name)
                       :src (quote ~body)
                       :type ::cgen}
                      ~cgen-fn))))

(defmacro cgen
  "Create a cgen (composite generator) - a composition of ugens and/or other
  cgens used as a general purpose synth building block.
  A cgen behaves similarly to a ugen - it has a default rate, named params, full
  docstring and may be mixed with both ugens and cgens in the construction of
  synths."
  [c-name summary doc params body rate categories]
  (let [c-name (symbol (name c-name))]
    (mk-cgen c-name summary doc params body categories rate)))
