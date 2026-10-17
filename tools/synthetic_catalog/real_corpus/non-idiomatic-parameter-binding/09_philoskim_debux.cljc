(ns debux.common.util
  "Utilities common for Clojure and ClojureScript"
  (:require [clojure.string :as str]
            [clojure.pprint :as pp]
            [clojure.zip :as z]
            [clojure.walk :as walk]))

(def ^:dynamic *indent-level* 0)

(def config*
  (atom {:debug-mode true
         :source-info-mode true
         :print-length 100
         :ns-blacklist nil
         :ns-whitelist nil
         :line-bullet "|"
         :cljs-devtools nil
         :tap-output false
         :date-time-fn nil}))

(defn println-with-flush [s]
  (println s)
  (flush))

(defn truncate [s]
  (if (> (count s) 70)
    (str (subs s 0 70) " ...")
    s))

(defn make-bullets
  [indent-level]
  (apply str (repeat indent-level (:line-bullet @config*))))

(defn prepend-bullets
  [line indent-level]
  (str (make-bullets indent-level) " " line))

(defn print-form-with-indent
  [form]
  (let [form' (prepend-bullets form *indent-level*)]
    (println-with-flush form')
    (when (:tap-output @config*)
      (tap> form') )))

(defn form-header [form & [msg]]
  (str (truncate (pr-str form))
       (and msg (str "   <" msg ">"))
       " =>"))

(defn pprint-result-with-indent
  [result]
  (let [pprint (str/trim (with-out-str (pp/pprint result)))
        prefix (str (make-bullets *indent-level*) "   ")
        result' (->> (str/split pprint #"\n")
                     (mapv #(str prefix %))
                     (str/join "\n"))]
    (println-with-flush result')
    (when (:tap-output @config*)
      (tap> result') )))

(defn spy-first
  [pre-result quoted-form & [opts]]
  (print-form-with-indent (form-header quoted-form))
  (pprint-result-with-indent pre-result)
  pre-result)
