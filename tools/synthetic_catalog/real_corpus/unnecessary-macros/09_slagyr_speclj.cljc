(ns speclj.core
  "Speclj's API. It contains nothing but macros, so that it can be used
  in both Clojure and ClojureScript."
  (:require [clojure.data]
            [clojure.string]
            [speclj.components]
            [speclj.config]
            [speclj.error]
            #?(:cljs    [speclj.platform]
               :default [speclj.platform :refer [if-cljs try-catch-anything]])
            [speclj.reporting]
            [speclj.results]
            [speclj.running]
            [speclj.stub]
            [speclj.tags]
            [speclj.report.documentation]
            [speclj.report.progress]
            [speclj.report.silent]
            [speclj.run.standard]))

(defn ^:no-doc -capture-loc
  "Returns a {:file :line :column} loc map from a form's reader metadata.
   Called at macroexpand time with the user-facing macro's &form and *file*.
   Returns nil when no line metadata is available.

   File resolution order:
   1. :file in the form's reader metadata (the cljs reader sets this)
   2. cljs.analyzer/*cljs-file* if bound (cljs compiler context)
   3. the supplied `file` argument (clojure.core/*file* at macroexpand time)

   On Clojure JVM and babashka, *file* is the user's source file. Under cljs,
   clojure.core/*file* is \"NO_SOURCE_PATH\" so we rely on (1) or (2)."
  [form file]
  (let [{:keys [line column] m-file :file} (meta form)]
    (when line
      (let [file (or m-file
                     (when-let [v (resolve 'cljs.analyzer/*cljs-file*)]
                       @v)
                     file)]
        {:file file :line line :column column}))))

(defmacro ^:no-doc -new-failure [message]
  `(ex-info ~message (merge {:type speclj.error/failure}
                            speclj.components/*source-loc*)))

(defmacro ^:no-doc -new-pending [message]
  `(ex-info ~message (merge {:type speclj.error/pending}
                            speclj.components/*source-loc*)))

(defmacro ^:no-doc help-it [name focused? & body]
  (let [loc (-capture-loc &form *file*)]
    (if (seq body)
      `(speclj.components/new-characteristic ~name nil (fn [] ~@body) ~focused? ~loc)
      ;; Body-less it/focus-it: synthesize a (pending) call carrying the user's
      ;; call-site metadata so the pending location reports the it/focus-it line
      ;; instead of sci internals.
      (let [pending-form (with-meta `(pending) (meta &form))]
        `(speclj.components/new-characteristic ~name nil (fn [] ~pending-form) ~focused? ~loc)))))

(defmacro ^:no-doc help-describe [name focused? & components]
  (let [loc (-capture-loc &form *file*)]
    `(let [description# (speclj.components/new-description ~name ~focused? ~(clojure.core/name (speclj.platform/get-name *ns*)) ~loc)]
       (binding [speclj.config/*parent-description* description#]
         ; MDM - use a vector below - cljs generates a warning because def/declares don't eval immediately
         (doseq [component# (vector ~@components)]
           (speclj.components/install component# description#)))
       (when-not (if-cljs
                   speclj.config/*parent-description*
                   (bound? #'speclj.config/*parent-description*))
         (speclj.running/submit-description (speclj.config/active-runner) description#))
       description#)))

(defmacro ^:no-doc with-source-loc
  "Binds speclj.components/*source-loc* to `loc` for the duration of `body` so
   any failure thrown inside can embed the location in its ex-data. Unlike
   help-should this does not increment the assertion counter; it is intended for
   wrapping macros (e.g. the 3-arg should-throw) that delegate to other should-*
   macros which already count their own assertions."
  [loc & body]
  `(binding [speclj.components/*source-loc* ~loc]
     ~@body))

(defmacro ^:no-doc help-should
  "Wraps a should-* expansion. The first argument is a {:file :line :column}
   loc map captured at the user-facing macro's call site (or nil). It is bound
   to speclj.components/*source-loc* so any failure thrown inside `body` can
   embed it in its ex-data."
  [loc & body]
  `(do (speclj.components/inc-assertions!)
       (with-source-loc ~loc ~@body)))

(defmacro it
  "body => any forms, but should contain at least one assertion (should)

  Declares a new characteristic (example in rspec)."
  [name & body]
  ;; Propagate the user's &form meta to help-it so its body-less branch can
  ;; attach it to the synthesized (pending) call.
  (with-meta `(help-it ~name false ~@body) (meta &form)))

(defmacro focus-it
  "Same as 'it', but it is meant to facilitate temporary debugging.
  Characteristics defined with this macro will be executed along with any
  other characteristics thus defined, but all other characteristic defined
  with 'it' will be ignored."
  [name & body]
  (with-meta `(help-it ~name true ~@body) (meta &form)))

(defmacro describe
  "body => & spec-components

  Declares a new spec.  The body can contain any forms that evaluate to spec components (it, before, after, with ...)."
  [name & components]
  ;; Propagate user's &form meta to help-describe so -capture-loc sees the user's line.
  (with-meta `(help-describe ~name false ~@components) (meta &form)))

(defmacro context
  "Same as describe, but should be used to nest testing contexts inside the outer describe.
  Contexts can be nested any number of times."
  [name & components]
  (with-meta `(describe ~name ~@components) (meta &form)))

(defmacro before
  "Declares a function that is invoked before each characteristic in the containing describe scope is evaluated. The body
  may consist of any forms, presumably ones that perform side effects."
  [& body]
  `(speclj.components/new-before (fn [] ~@body)))

(defmacro after
  "Declares a function that is invoked after each characteristic in the containing describe scope is evaluated. The body
  may consist of any forms, presumably ones that perform side effects."
  [& body]
  `(speclj.components/new-after (fn [] ~@body)))

(defn ^:no-doc -make-with [name body ctor bang?]
  (let [var-name (with-meta (symbol name) {:dynamic true})]
    `(do
       (declare ~var-name)
       (~ctor '~var-name (fn [] ~@body) (fn [v#] (set! ~var-name v#)) ~bang?))))

(defmacro with
  "Declares a reference-able symbol that will be lazily evaluated once per characteristic of the containing
  describe scope.  The body may contain any forms, the last of which will be the value of the dereferenced symbol.

  (with meaning 42)
  (it \"knows the meaning of life\" (should= @meaning (the-meaning-of :life)))"
  [name & body]
  (-make-with name body `speclj.components/new-with false))

(defmacro ^:no-doc -to-s [thing]
  `(if-some [thing# ~thing] (pr-str thing#) "nil"))

(defmacro -fail
  "Useful for making custom assertions."
  [message]
  (let [loc (-capture-loc &form *file*)]
    `(throw (ex-info ~message (merge {:type speclj.error/failure}
                                     ~loc
                                     speclj.components/*source-loc*)))))

(defmacro should
  "Asserts the truthy-ness of a form"
  [form]
  `(help-should ~(-capture-loc &form *file*)
     (let [value# ~form]
       (when-not value#
         (-fail (str "Expected truthy but was: " (-to-s value#) ""))))))

(defmacro should=
  "Asserts that two forms evaluate to equal values, with the expected value as the first parameter."
  ([expected-form actual-form]
   `(help-should ~(-capture-loc &form *file*)
      (let [expected# ~expected-form actual# ~actual-form]
        (when-not (= expected# actual#)
          (-fail (str "Expected: " (-to-s expected#) speclj.platform/endl "     got: " (-to-s actual#) " (using =)"))))))
  ([expected-form actual-form delta-form]
   `(help-should ~(-capture-loc &form *file*)
      (let [expected# ~expected-form
            actual#   ~actual-form
            delta#    ~delta-form]
        (when (speclj.platform/difference-greater-than-delta? expected# actual# delta#)
          (-fail (str "Expected: " (-to-s expected#) speclj.platform/endl "     got: " (-to-s actual#) " (using delta: " delta# ")")))))))

(defmacro should-not=
  "Asserts that two forms evaluate to unequal values, with the unexpected value as the first parameter."
  [expected-form actual-form]
  `(help-should ~(-capture-loc &form *file*)
     (let [expected# ~expected-form actual# ~actual-form]
       (when (= expected# actual#)
         (-fail (str "Expected: " (-to-s expected#) speclj.platform/endl "not to =: " (-to-s actual#)))))))

(defmacro should-not-be-nil
  "Asserts that the form evaluates to a non-nil value"
  [form]
  ;; Delegate to should-not=, propagating user's &form meta.
  (with-meta `(should-not= nil ~form) (meta &form)))

(defmacro ^:no-doc -create-should-throw-failure [expected actual expr]
  `(let [expected-name# (speclj.platform/type-name ~expected)
         expected-gaps# (apply str (repeat (count expected-name#) " "))
         actual-string# (if-let [actual# ~actual] (pr-str actual#) "<nothing thrown>")]
     (-new-failure (str "Expected " expected-name# " thrown from: " (pr-str ~expr) speclj.platform/endl
                        "         " expected-gaps# "     but got: " actual-string#))))

(defmacro should-throw
  "Asserts that a Throwable is throws by the evaluation of a form.
When a class is passed, it asserts that the thrown Exception is an instance of the class.
There are three options for passing different kinds of predicates:
  - If a string, assert that the message of the Exception is equal to the string.
  - If a regex, asserts that the message of the Exception matches the regex.
  - If a function, assert that calling the function on the Exception returns a truthy value."
  ([form]
   ;; Delegate to 2-arg should-throw, propagating user's &form meta.
   (with-meta `(should-throw speclj.platform/throwable ~form) (meta &form)))
  ([throwable-type form]
   `(help-should ~(-capture-loc &form *file*)
      (try-catch-anything
        ~form
        (throw (-create-should-throw-failure ~throwable-type nil '~form))
        (catch e#
               (cond
                 (speclj.error/failure? e#) (throw e#)
                 (not (instance? ~throwable-type e#)) (throw (-create-should-throw-failure ~throwable-type e# '~form))
                 :else e#)))))
  ([throwable-type predicate form]
   ;; Bind *source-loc* (without bumping the assertion counter — the inner
   ;; should-throw and should=/should-not-be-nil delegations below already count
   ;; their own assertions) so the user's (should-throw ...) site is reported
   ;; for any failure thrown by those nested calls or by -fail in the catch.
   `(with-source-loc ~(-capture-loc &form *file*)
      (let [e# (should-throw ~throwable-type ~form)]
        (try-catch-anything
          (let [predicate# ~predicate]
            (cond (speclj.platform/re? predicate#)
                  (should-not-be-nil (re-find predicate# (speclj.platform/error-message e#)))

                  (ifn? predicate#)
                  (should= true (predicate# e#))

                  :else
                  (should= predicate# (speclj.platform/error-message e#))))

          (catch f# (-fail (str "Expected exception predicate didn't match" speclj.platform/endl (speclj.platform/error-message f#)))))))))

(defmacro pending
  "When added to a characteristic, it is marked as pending.  If a message is provided it will be printed
  in the run report"
  ([] (with-meta `(pending "Not Yet Implemented") (meta &form)))
  ([message]
   `(throw (ex-info ~message (merge {:type speclj.error/pending}
                                    ~(-capture-loc &form *file*))))))
