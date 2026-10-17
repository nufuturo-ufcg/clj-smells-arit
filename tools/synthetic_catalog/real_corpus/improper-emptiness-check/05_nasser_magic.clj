(ns magic.analyzer.loop-bindings
  (:require
   [clojure.set :as s]
   [clojure.tools.analyzer.ast :as ast]
   [magic.analyzer.types :refer [ast-type numeric-type? best-numeric-promotion]]
   [magic.analyzer.generated-types :refer [*reusable-types*]]
   [magic.emission :refer [*module*]]))

(defn type-depth
  ([t] (type-depth t 0))
  ([t d] (if (or (nil? t) (= t Object) (.IsInterface t))
           d
           (recur (.BaseType t) (inc d)))))

(defn best-type [a b]
  (if (and (numeric-type? a) (numeric-type? b))
    (best-numeric-promotion [a b])
    (let [common-ancestors   (s/intersection
                              (conj (or (ancestors a) #{}) a)
                              (conj (or (ancestors b) #{}) b))
          common-types       (remove #(.IsInterface %) common-ancestors)
          common-interfaces  (filter #(.IsInterface %) common-ancestors)
          deepest-base-type  (when-not (empty? common-types)
                               (reduce
                                (fn [t1 t2]
                                  (if (> (type-depth t2) (type-depth t1)) t2 t1))
                                common-types))
          deepest-base-type  (if (= deepest-base-type ValueType) Object deepest-base-type)
          heaviest-interface (when-not (empty? common-interfaces)
                               (reduce
                                (fn [t1 t2]
                                  (if (> (count (.GetInterfaces t2))
                                         (count (.GetInterfaces t1)))
                                    t2 t1))
                                common-interfaces))]
      (or deepest-base-type heaviest-interface Object))))
