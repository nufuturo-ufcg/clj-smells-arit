(defn- expand-method-impl-cache [^clojure.lang.MethodImplCache cache c f]
  (if (.map cache)
    (let [cs (assoc (.map cache) c (clojure.lang.MethodImplCache+Entry. c f))]                            ;;; clojure.lang.MethodImplCache$Entry
      (clojure.lang.MethodImplCache. (.protocol cache) (.methodk cache) cs))
    (let [cs (into1 {} (remove (fn [[c e]] (nil? e)) (map vec (partition 2 (.table cache)))))
          cs (assoc cs c (clojure.lang.MethodImplCache+Entry. c f))]                                      ;;; clojure.lang.MethodImplCache$Entry
      (if-let [[shift mask] (maybe-min-hash (map hash (keys cs)))]
        (let [table (make-array Object (* 2 (inc mask)))
              table (reduce1 (fn [^objects t [c e]]
                               (let [i (* 2 (int (shift-mask shift mask (hash c))))]
                                 (aset t i c)
                                 (aset t (inc i) e)
                                 t))
                             table cs)]
          (clojure.lang.MethodImplCache. (.protocol cache) (.methodk cache) shift mask table))
        (clojure.lang.MethodImplCache. (.protocol cache) (.methodk cache) cs)))))
