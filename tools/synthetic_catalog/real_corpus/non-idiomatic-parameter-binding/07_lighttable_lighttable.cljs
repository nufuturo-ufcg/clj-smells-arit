(ns lt.objs.context
  "Provide context object which manages temporary contexts LT can get in.
  An object can be associated with a context which is useful for keeping track
  of current tabset or browser"
  (:require [lt.object :as object])
  (:require-macros [lt.macros :refer [behavior]]))

(def contexts (atom #{}))

(def ctx->obj (atom {}))

(def ctx->group (atom {}))

(def group->ctxs (atom {}))

(defn out!
  ([ctxs]
   (let [ctxs (if (coll? ctxs)
                ctxs
                [ctxs])]
     (swap! contexts #(apply disj % ctxs))
     (swap! ctx->obj #(apply dissoc % ctxs))
     (object/raise ctx-obj :log!)))
   ([ctxs _] (out! ctxs)))

(defn in! [ctxs & [obj]]
  (let [ctxs (if (coll? ctxs)
               ctxs
               [ctxs])]
    (swap! contexts #(apply conj % ctxs))
    (swap! ctx->obj #(merge % (zipmap ctxs (repeat obj))))
    (object/raise ctx-obj :log!)
    (doseq [c ctxs]
      (when-let [group (@group->ctxs (@ctx->group c))]
        (out! group)))))

(def ctx-obj (object/create ::context))
