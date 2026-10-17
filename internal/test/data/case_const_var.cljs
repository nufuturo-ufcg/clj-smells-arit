(ns case-const-cljs)
(def ^:const FILL 0)
(defn f [x] (case x
  FILL :fill
  :other))
