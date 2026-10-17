(ns karabiner-configurator.rules
  (:require
   [karabiner-configurator.conditions :as conditions]
   [karabiner-configurator.data :as d :refer [pkey? k? special-modi-k? conf-data pointing-k? consumer-k? noti? softf? templates? devices? assoc-conf-data update-conf-data profile? raw-rule?]]
   [karabiner-configurator.froms :as froms]
   [karabiner-configurator.misc :refer [massert contains??]]
   [karabiner-configurator.tos :as tos]))

(defn merge-multiple-device-conditions
  [device-vec]
  (assoc-conf-data :devices (dissoc (:devices @conf-data) :temp-device))
  (let [devices-list
        (for [item device-vec
              :when (and (keyword? item) (devices? item))
              :let [this-device-vec (item (:devices @conf-data))
                    temp-device-vec (if (devices? :temp-device)
                                      (vec (concat (:temp-device (:devices @conf-data)) this-device-vec))
                                      this-device-vec)
                    _update-temp-device-into-conf-data (update-conf-data (assoc-in @conf-data [:devices :temp-device] temp-device-vec))]]
          item)
        use-temp-device? (pos? (count devices-list))
        new-conditions (if use-temp-device? (conj (vec (reduce #(remove #{%2} %1) device-vec devices-list)) :temp-device)
                           device-vec)]
    new-conditions))

(def profile-layer-condis (atom {}))

(defn conditions-key
  [_des conditions prev-result profiles]
  (let [conditions (merge-multiple-device-conditions conditions)
        result-condi
        (if (conditions/is-simple-set-variable? conditions)
          {:conditions (conditions/parse-conditions [conditions] (:from prev-result) (:to prev-result))}
          {:conditions (conditions/parse-conditions conditions (:from prev-result) (:to prev-result))})
        layer-condis (filter #(:is-layer (meta %)) (:conditions result-condi))]
    (when (pos? (count layer-condis))
      (doseq [profile profiles]
        (doseq [layer-condi layer-condis]
          (when (not (contains?? (profile @profile-layer-condis) (keyword (:name layer-condi))))
            (reset! profile-layer-condis
                    (assoc @profile-layer-condis profile (conj (or (profile @profile-layer-condis) []) (keyword (:name layer-condi)))))))))
    result-condi))
