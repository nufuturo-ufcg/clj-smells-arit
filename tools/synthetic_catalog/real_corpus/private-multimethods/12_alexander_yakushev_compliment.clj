(ns compliment.sources.class-members
  "Completion for both static and non-static class members."
  (:require [clojure.string :as str :refer [join]]
            [compliment.sources :refer [defsource]]
            [compliment.sources.local-bindings :refer [bindings-from-context]]
            [compliment.utils :as utils :refer [fuzzy-matches-no-skip?
                                                resolve-class]])
  (:import [java.lang.reflect Field Member Method Modifier Constructor Executable]))

(defn type-to-pretty-string
  "Takes a type (either a class or a primitive) and returns it's
  human-readable name."
  [^Class t]
  (if (or (.isLocalClass t)
          (.isMemberClass t))
    (.getName t)
    (.getSimpleName t)))

(defn doc-method-parameters
  "Takes a list of method parameters and stringifies it."
  [parameters]
  (->> parameters
       (map type-to-pretty-string)
       (interpose " ")
       join
       (format "(%s)")))

(defn- members->doc-dispatch [[^Class _cl members]]
  (class (first members)))

(defmulti ^:private members->doc #'members->doc-dispatch)

(defmethod members->doc java.lang.reflect.Method
  [[^Class cl members]]
  (let [^Member f-mem (first members)]
    (str (.getName cl) "." (.getName f-mem)
         (join
          (map (fn [^Method member]
                 (str "\n  " (doc-method-parameters (.getParameterTypes member))
                      " -> " (type-to-pretty-string (.getReturnType ^Method member))
                      " (" (Modifier/toString (.getModifiers member)) ")"))
               members))
         "\n")))

(defmethod members->doc java.lang.reflect.Constructor
  [[^Class cl members]]
  (str (.getName cl) ".new"
       (join
        (map (fn [^Constructor member]
               (str "\n  " (doc-method-parameters (.getParameterTypes member))))
             members))
       "\n"))

(defmethod members->doc java.lang.reflect.Field
  [[^Class cl members]]
  (let [^Member f-mem (first members)]
    (str (.getName cl) "." (.getName f-mem)
         " = " (try (.get ^Field f-mem nil)
                    (catch Exception _e "?"))
         " (" (type-to-pretty-string (.getType ^Field f-mem)) ")\n"
         (Modifier/toString (.getModifiers f-mem))
         "\n")))
