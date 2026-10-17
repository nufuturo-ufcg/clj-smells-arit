(ns hx.hooks
  (:require ["react" :as react]
            [goog.object :as gobj]))

(defn useIRef
  "Takes an initial value. Returns an atom that will _NOT_ re-render component
  on change."
  [initial]
  (let [sentinel (react/useRef #js {})]
    (react/useMemo
     (fn []
       (specify! #js {:current initial}
         IDeref
         (-deref [this]
           (.-current ^js this))

         IReset
         (-reset! [this v]
           (gobj/set this "current" v))

         ISwap
         (-swap!
           ([this f]
            (gobj/set this "current" (f (.-current ^js this))))
           ([this f a]
            (gobj/set this "current" (f (.-current ^js this) a)))
           ([this f a b]
            (gobj/set this "current" (f (.-current ^js this) a b)))
           ([this f a b xs]
            (gobj/set this "current" (apply f (.-current ^js this) a b xs))))))
     #js [sentinel])))

(defn useMemo
  "Like react/useMemo, but deps can be a CLJS collection and is optional."
  ([f]
   (react/useMemo f #js []))
  ([f deps]
   (react/useMemo f (to-array deps))))
