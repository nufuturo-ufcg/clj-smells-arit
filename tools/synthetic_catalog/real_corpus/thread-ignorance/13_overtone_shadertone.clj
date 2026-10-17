(ns #^{:author "Roger Allen"
       :doc "Shadertone lisp-to-glsl translation."}
  shadertone.translate
  (:require [clojure.walk :as walk]
            [clojure.string :as string]))

(defn- shader-walk-defn-args [x]
  ;;(println "shader-walk-defn-args-0" x (empty? x))
  (assert (vector? x))
  (if (empty? x)
    "void"
    (string/join \, (map #(apply (partial format "%s %s") %) (partition 2 x)))))
