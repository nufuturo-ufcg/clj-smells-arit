(ns eastwood.util
  (:require
   [clojure.java.io :as io]
   [clojure.pprint :as pp]
   [clojure.repl :as repl]
   [clojure.set :as set]
   [clojure.string :as string]
   [clojure.walk :as walk]
   [eastwood.copieddeps.dep1.clojure.tools.analyzer.ast :as ast]
   [eastwood.copieddeps.dep10.clojure.tools.reader :as reader]
   [eastwood.copieddeps.dep10.clojure.tools.reader.reader-types :as reader-types])
  (:import
   (clojure.lang LineNumberingPushbackReader)
   (java.io File StringReader)))

(defn has-keys? [m key-seq]
  (every? #(contains? m %) key-seq))

(defn sorted-map-with-non-keyword-keys? [x]
  (and (map? x)
       (sorted? x)
       (some (fn [o] (not (keyword? o))) (keys x))))

(defn protocol?
  "Make a good guess as to whether p is an object created via defprotocol."
  [p]
  (and (map? p)
       ;; Don't even try to do has-keys? on a sorted map with
       ;; non-keyword keys, since it will most likely cause compare to
       ;; throw an exception.
       (not (sorted-map-with-non-keyword-keys? p))
       (has-keys? p [:on :on-interface :sigs :var :method-map
                     :method-builders])))

(defn map-vals [f m]
  (into (empty m)
        (for [[k v] m] [k (f v)])))

(def loaded-but-not-in-file-marker "        Loaded but not in file:")

(defn print-var-info-summary [var-info-map]
  (let [file-vars-by-ns (->> (keys var-info-map)
                             (group-by #(namespace %))
                             (map-vals #(set (map name %))))
        ;; Try to require the namespaces mentioned in the file, but
        ;; mask any exceptions that occur when trying.
        ;; TBD: figure out extra namespaces from each .jar that may not be reflected in var-info.edn
        ;; ...use simply (all-ns) after performing a first batch of `require`s
        _ (doseq [ns-name (keys file-vars-by-ns)]
            (try
              (println "require " ns-name "...")
              (require (symbol ns-name))
              (catch Throwable _)))
        loaded-vars-by-ns (->> (all-ns)
                               (map (fn [ns]
                                      [(str ns)
                                       (->> (ns-publics ns)
                                            keys
                                            (map str)
                                            set)]))
                               (into {}))
        ns-names (into (sorted-set) (concat (keys file-vars-by-ns)
                                            (keys loaded-vars-by-ns)))
        ;; returns a handy string that can easily be copied into var-info.edn:
        template (fn [ns-name var-name]
                   (let [fqs (symbol (str ns-name)
                                     (str var-name))
                         var-ref (when (-> ns-name str symbol find-ns)
                                   (find-var fqs))
                         macro? (if (-> var-ref meta :macro)
                                  true
                                  nil)
                         ctor? (or (->> var-name str (re-find #"^->[A-Z]"))
                                   (and (->> var-name str (re-find #"^map->[A-Z]"))
                                        (let [fqs (symbol (str ns-name)
                                                          (-> var-name str (string/replace #"^map" "")))]
                                          (try
                                            (find-var fqs)
                                            (catch Exception _
                                              nil))))
                                   (->> var-name str (re-find #"^make-"))
                                   (->> var-name str (re-find #"-factory$")))
                         predicate? (-> var-name str (.endsWith "?"))
                         uninteresting? (or (some-> var-ref deref protocol?)
                                            (and var-ref
                                                 (or (nil? @var-ref)
                                                     (instance? clojure.lang.Var$Unbound @var-ref)
                                                     (coll? @var-ref)
                                                     (not (ifn? @var-ref)))))]
                     (cond
                       uninteresting?
                       (format " %s {:var-kind nil, :macro nil}"
                               fqs)

                       ctor?
                       (format " %s {:var-kind nil, :macro %s, :predicate %s, :side-effect false, :pure-fn true, :warn-if-ret-val-unused true, :lazy false, :pure-if-fn-args-pure true, :io-fn false, :evals-exprs false}"
                               fqs
                               (pr-str macro?)
                               predicate?)

                       :else
                       ;; this format includes all keys, so as to invite maintainers to think of all relevant info.
                       (format " %s {:var-kind nil, :macro %s, :predicate %s, :side-effect , :pure-fn , :warn-if-ret-val-unused , :lazy , :pure-if-fn-args-pure , :io-fn , :evals-exprs }"
                               fqs
                               (pr-str macro?)
                               predicate?))))]
    (println "Clojure version " (clojure-version))
    (println)
    (println "Summary of contents of var-info.edn file:")
    (println)
    (println "Columns:")
    (println "A  # of symbols in the file")
    (println "B  namespace is loaded?")
    (println "C  # of symbols in loaded namespace, but not in the file")
    (println "D  # of symbols in the file, but not in the loaded namespace")
    (println)
    (println (format "   A B    C    D Namespace name"))
    (doseq [ns-name ns-names]
      (let [loaded-vars (get loaded-vars-by-ns ns-name)
            file-vars (get file-vars-by-ns ns-name #{})
            loaded-but-not-file (set/difference (or loaded-vars #{}) file-vars)
            file-but-not-loaded (set/difference file-vars (or loaded-vars #{}))]
        (println
         (format "%4d %s %4s %4s %s"
                 (count file-vars)
                 (if loaded-vars "Y" "N")
                 (if loaded-vars
                   (count loaded-but-not-file)
                   "-")
                 (if loaded-vars
                   (count file-but-not-loaded)
                   "-")
                 ns-name))
        (when (> (count file-vars) 0)
          (when (> (count loaded-but-not-file) 0)
            (println (format loaded-but-not-in-file-marker))
            (doseq [name (sort loaded-but-not-file)]
              (println (template ns-name name))))
          (when (> (count file-but-not-loaded) 0)
            (println (format "            In file but not loaded:"))
            (doseq [name (sort file-but-not-loaded)]
              (println (format "            %s" name)))))))))
