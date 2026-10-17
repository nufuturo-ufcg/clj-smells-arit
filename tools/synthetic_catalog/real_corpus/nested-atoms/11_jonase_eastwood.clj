(ns eastwood.copieddeps.dep1.clojure.tools.analyzer.passes.add-binding-atom
  (:require [eastwood.copieddeps.dep1.clojure.tools.analyzer.ast :refer [prewalk]]
            [eastwood.copieddeps.dep1.clojure.tools.analyzer.passes.uniquify :refer [uniquify-locals]]))

(defn add-binding-atom
  "Adds an atom-backed-map to every local binding,the same
   atom will be shared between all occurences of that local.

   The atom is put in the :atom field of the node."
  {:pass-info {:walk :pre :depends #{#'uniquify-locals} :state (fn [] (atom {}))}}
  ([ast] (prewalk ast (partial add-binding-atom (atom {}))))
  ([state ast]
     (case (:op ast)
       :binding
       (let [a (atom {})]
         (swap! state assoc (:name ast) a)
         (assoc ast :atom a))
       :local
       (if-let [a (@state (:name ast))]
         (assoc ast :atom a)
         ;; handle injected locals
         (let [a (get-in ast [:env :locals (:name ast) :atom] (atom {}))]
           (swap! state assoc (:name ast) a)
           (assoc ast :atom a)))
       ast)))
