(ns cloverage.coverage
  (:gen-class)
  (:require [clojure.java.classpath :as cp]
            [clojure.java.io :as io]
            [clojure.test :as test]
            [clojure.test.junit :as junit]
            [clojure.tools.namespace.find :as ns-find]
            [cloverage.args :as args]
            [cloverage.debug :as debug]
            [cloverage.dependency :as dep]
            [cloverage.instrument :as inst]
            [cloverage.report :as rep]
            [cloverage.report.codecov :as codecov]
            [cloverage.report.console :as console]
            [cloverage.report.coveralls :as coveralls]
            [cloverage.report.emma-xml :as emma-xml]
            [cloverage.report.html :as html]
            [cloverage.report.lcov :as lcov]
            [cloverage.report.raw :as raw]
            [cloverage.report.text :as text]
            [cloverage.source :as src])
  (:import java.io.FileNotFoundException
           java.util.concurrent.atomic.AtomicInteger))

(def ^:dynamic *covered* (atom []))

(defn form-for-suppressing-unselected-tests
  [namespaces selectors test-func]
  (let [copy-meta (fn [var from-key to-key]
                    (when-let [x (get (meta var) from-key)]
                      (alter-meta! var #(-> % (assoc to-key x) (dissoc from-key)))))
        vars      (when (seq selectors)
                    ;; need to manually require namespaces to make sure they've been
                    ;; loaded before test runner has a chance to run
                    (doseq [n namespaces]
                      (require n))
                    (->> namespaces
                         (map ns-interns)
                         (mapcat vals)
                         (remove (fn [var]
                                   (some (fn [selector]
                                           (let [meta-info (merge (-> var meta :ns meta)
                                                                  (assoc (meta var) :leiningen.test/var var))]
                                             (selector meta-info)))
                                         selectors)))))
        copy #(doseq [v vars]
                (copy-meta v %1 %2))]
    (copy :test :leiningen/skipped-test)
    (try (test-func)
         (finally
           (copy :leiningen/skipped-test :test)))))

(defn covered []
  (mapv (fn [{:keys [^AtomicInteger hits] :as form}]
          (merge form {:hits (.get hits) :covered (pos? (.get hits))}))
        @*covered*))

(defn find-nses
  "Given ns-paths and regex-patterns returns:
  * empty sequence when ns-paths is empty and regex-patterns is empty
  * all namespaces on all ns-paths (if regex-patterns is empty)
  * all namespaces on the classpath that match any of the regex-patterns (if ns-paths is empty)
  * namespaces on ns-paths that match any of the regex-patterns"
  [ns-paths regex-patterns]
  (debug/tprn "find:" {:ns-paths       ns-paths
                       :regex-patterns regex-patterns})
  (let [namespaces (map name
                        (cond
                          (and (empty? ns-paths) (empty? regex-patterns))
                          '()

                          (empty? ns-paths)
                          (distinct (ns-find/find-namespaces (filter #(.isDirectory ^java.io.File %)
                                                                     (cp/system-classpath))))

                          :else
                          (distinct (ns-find/find-namespaces (map io/file ns-paths)))))]
    (debug/tprn "found:" {:namespaces     namespaces
                          :ns-paths       ns-paths
                          :regex-patterns regex-patterns})
    (if (seq regex-patterns)
      (filter (fn [ns] (some #(re-matches % ns) regex-patterns)) namespaces)
      namespaces)))

(defn- resolve-var [sym]
  (let [ns (namespace (symbol sym))
        ns (when ns (symbol ns))]
    (when ns
      (require ns))
    (ns-resolve (or ns *ns*)
                (symbol (name sym)))))

(defn- require-ns
  "Require and return namespace by given string."
  [ns-str]
  (let [ns-sym (symbol ns-str)]
    (require ns-sym)
    (find-ns ns-sym)))

(defmulti runner-fn :runner)

(defmethod runner-fn :midje [_]
  (if-let [f (resolve-var 'midje.repl/load-facts)]
    (fn [nses]
      {:errors (:failures (apply f nses))})
    (throw (RuntimeException. "Failed to load Midje."))))

(defmethod runner-fn :clojure.test [{:keys [junit? output] :as _opts}]
  (fn [nses]
    (let [run-tests (fn []
                      (apply require (map symbol nses))
                      {:errors (reduce + ((juxt :error :fail)
                                          (apply test/run-tests nses)))})]
      (if junit?
        (do
          (.mkdirs (io/file output))
          (binding [test/*test-out* (io/writer (io/file output "junit.xml"))]
            (junit/with-junit-output (run-tests))))
        (run-tests)))))

(defmethod runner-fn :eftest
  [{:keys [runner-opts] :as opts}]
  (fn [nses]
    (let [find-tests-fn (resolve-var 'eftest.runner/find-tests)
          run-tests-fn (resolve-var 'eftest.runner/run-tests)
          eftest-opts (if (and (seq runner-opts)
                               (not (map? runner-opts)))
                        (into {} runner-opts)
                        runner-opts)
          eftest-opts (cond-> eftest-opts
                        (contains? eftest-opts :report)
                        (update :report resolve-var))
          extra-test-ns (or (seq (:extra-test-ns opts)) [])
          test-namespaces (->> (find-nses (:test-ns-path opts) (:test-ns-regex opts))
                               (concat extra-test-ns)
                               (set)
                               (map require-ns)
                               (seq))
          test-vars (find-tests-fn test-namespaces)
          results (run-tests-fn test-vars eftest-opts)]
      (apply require (map symbol nses))
      {:errors (reduce + ((juxt :error :fail) results))})))

(defmethod runner-fn :default [_]
  (throw (IllegalArgumentException.
          "Runner not found. Built-in runners are `clojure.test`, `midje` and `eftest`.")))

(defn run-tests [{:keys [runner test-selectors selector junit?], :as opts} test-nses]
  ;; load runner multimethod definition from other dependencies
  (when-not (#{:clojure.test :midje :eftest} runner)
    (try (require (symbol (format "%s.cloverage" (name runner))))
         (catch FileNotFoundException _)))
  (let [test-result (when (seq test-nses)
                      (if (and junit? (not= runner :clojure.test))
                        (throw (RuntimeException.
                                "Junit output only supported for clojure.test at present"))
                        (let [test-ns-symbols (map symbol test-nses)]
                          (form-for-suppressing-unselected-tests test-ns-symbols
                                                                 (vals (select-keys test-selectors selector))
                                                                 #((runner-fn opts) test-ns-symbols)))))
        forms       (rep/gather-stats (covered))
        ;; sum up errors as in lein test
        num-errors  (when test-result
                      (:errors test-result))]
    (println "Ran tests.")
    {:test-result test-result, :forms forms, :num-errors num-errors}))
