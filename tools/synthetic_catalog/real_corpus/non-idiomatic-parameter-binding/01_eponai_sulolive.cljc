(ns eponai.common.ui.elements.input-validate
  (:require
    #?(:cljs [cljs.spec.alpha :as s]
       :clj [clojure.spec.alpha :as s])
            [eponai.common.ui.dom :as my-dom]
            [eponai.common.ui.elements.css :as css]
            [taoensso.timbre :refer [debug]]))

(defn validate
  [spec m form-inputs & [prefix-key]]
  (when-let [err (s/explain-data spec m)]
    (let [problems (::s/problems err)
          invalid-paths (map (fn [p]
                               (str prefix-key (get form-inputs (last p))))
                             (into (map :path problems) (map :via problems)))]
      {:explain-data  err
       :invalid-paths (set invalid-paths)})))
