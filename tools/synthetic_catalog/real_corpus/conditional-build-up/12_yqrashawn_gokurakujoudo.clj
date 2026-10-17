(ns karabiner-configurator.modifiers
  (:require [karabiner-configurator.data :as d]
            [karabiner-configurator.misc :refer [massert]]))

(defn parse-modifier-arr-or-keyword
  [modi modifier-name]
  (cond (keyword? modi)
        (do (massert (d/modifier-k? modi) (str "invalid modifier " modi " in " modifier-name))
            [(name modi)])
        (vector? modi)
        (mapv
         (fn [vec]
           (massert (d/modifier-k? vec) (str "invalid modifier " vec " in " modifier-name))
           (name vec)) modi)))

(defn parse-map-modifiers
  [modifier-name modifier-info]
  (let [{:keys [mandatory optional]} modifier-info
        result {}
        result (if (some? mandatory)
                 (assoc result
                        :mandatory
                        (parse-modifier-arr-or-keyword mandatory modifier-name))
                 result)
        result (if (some? optional)
                 (assoc result
                        :optional
                        (parse-modifier-arr-or-keyword optional modifier-name)) result)]
    result))
