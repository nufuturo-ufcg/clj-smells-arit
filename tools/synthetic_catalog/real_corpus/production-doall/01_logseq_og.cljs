(ns frontend.util.property
  "Property fns needed by the rest of the app and not graph-parser"
  (:require [clojure.string :as string]
            [frontend.util :as util]
            [clojure.set :as set]
            [frontend.config :as config]
            [logseq.graph-parser.util :as gp-util]
            [logseq.graph-parser.mldoc :as gp-mldoc]
            [logseq.graph-parser.property :as gp-property :refer [properties-start properties-end]]
            [logseq.graph-parser.util.page-ref :as page-ref]
            [frontend.format.mldoc :as mldoc]
            [logseq.graph-parser.text :as text]
            [frontend.db :as db]
            [frontend.state :as state]
            [frontend.util.cursor :as cursor]
            [frontend.util.block-content :as content]))

(defn simplified-property?
  [line]
  (boolean
   (and (string? line)
        (re-find (re-pattern (str "^\\s?[^ ]+" gp-property/colons)) line))))

(defn front-matter-property?
  [line]
  (boolean
   (and (string? line)
        (util/safe-re-find #"^\s*[^ ]+:" line))))

(defn build-properties-str
  [format properties]
  (when (seq properties)
    (let [org? (= format :org)
          kv-format (if org? ":%s: %s" (str "%s" gp-property/colons " %s"))
          full-format (if org? ":PROPERTIES:\n%s\n:END:" "%s\n")
          properties-content (->> (map (fn [[k v]] (util/format kv-format (name k) v)) properties)
                                  (string/join "\n"))]
      (util/format full-format properties-content))))

(defn front-matter?
  [s]
  (string/starts-with? s "---\n"))

(defn insert-property
  "Only accept nake content (without any indentation)"
  ([format content key value]
   (insert-property format content key value false))
  ([format content key value front-matter?]
   (when (string? content)
     (let [ast (content/get-ast content format)
           title? (content/has-title? content format)
           has-properties? (or (and title?
                                    (or (mldoc/properties? (second ast))
                                        (mldoc/properties? (second
                                                            (remove
                                                             (fn [[x _]]
                                                               (contains? #{"Hiccup" "Raw_Html"} (first x)))
                                                             ast)))))
                               (mldoc/properties? (first ast)))
           lines (string/split-lines content)
           [title body] (content/get-title&body content format)
           scheduled (filter #(string/starts-with? % "SCHEDULED") lines)
           deadline (filter #(string/starts-with? % "DEADLINE") lines)
           body-without-timestamps (filter
                                    #(not (or (string/starts-with? % "SCHEDULED")
                                              (string/starts-with? % "DEADLINE")))
                                    (string/split-lines body))
           org? (= :org format)
           key (string/lower-case (name key))
           value (string/trim (str value))
           start-idx (.indexOf lines properties-start)
           end-idx (.indexOf lines properties-end)
           result (cond
                    (and org? (not has-properties?))
                    (let [properties (build-properties-str format {key value})]
                      (if title
                        (string/join "\n" (concat [title] scheduled deadline [properties] body-without-timestamps))
                        (str properties "\n" content)))

                    (and has-properties? (>= start-idx 0) (> end-idx 0) (> end-idx start-idx))
                    (let [exists? (atom false)
                          before (subvec lines 0 start-idx)
                          middle (doall
                                  (->> (subvec lines (inc start-idx) end-idx)
                                       (mapv (fn [text]
                                               (let [[k v] (gp-util/split-first ":" (subs text 1))]
                                                 (if (and k v)
                                                   (let [key-exists? (= k key)
                                                         _ (when key-exists? (reset! exists? true))
                                                         v (if key-exists? value v)]
                                                     (str ":" k ": "  (string/trim v)))
                                                   text))))))
                          middle (if @exists? middle (conj middle (str ":" key ": "  value)))
                          after (subvec lines (inc end-idx))
                          lines (concat before [properties-start] middle [properties-end] after)]
                      (string/join "\n" lines))

                    (not org?)
                    (let [exists? (atom false)
                          sym (if front-matter? ": " (str gp-property/colons " "))
                          new-property-s (str key sym value)
                          property-f (if front-matter? front-matter-property? simplified-property?)
                          groups (partition-by property-f lines)
                          compose-lines (fn []
                                          (mapcat (fn [lines]
                                                    (if (property-f (first lines))
                                                      (let [lines (doall
                                                                   (mapv (fn [text]
                                                                           (let [[k v] (gp-util/split-first sym text)]
                                                                             (if (and k v)
                                                                               (let [key-exists? (= k key)
                                                                                     _ (when key-exists? (reset! exists? true))
                                                                                     v (if key-exists? value v)]
                                                                                 (str k sym  (string/trim v)))
                                                                               text)))
                                                                         lines))
                                                            lines (if @exists? lines (conj lines new-property-s))]
                                                        lines)
                                                      lines))
                                                  groups))
                          lines (cond
                                  has-properties?
                                  (compose-lines)

                                  title?
                                  (cons (first lines) (cons new-property-s (rest lines)))

                                  :else
                                  (cons new-property-s lines))]
                      (string/join "\n" lines))

                    :else
                    content)]
       (string/trimr result)))))
