(ns lt.objs.style
  "Provide styling behaviors, themes and skins"
  (:require [lt.object :as object]
            [lt.objs.sidebar.command :as cmd]
            [lt.objs.app :as app]
            [lt.objs.tabs :as tabs]
            [lt.objs.context :as ctx]
            [lt.objs.plugins :as plugins]
            [lt.objs.editor :as editor]
            [lt.objs.editor.pool :as pool]
            [lt.objs.files :as files]
            [lt.objs.deploy :as deploy]
            [lt.util.dom :as dom]
            [lt.util.load :as load]
            [singultus.binding :refer [bound -value subatom]]
            [singultus.compiler :refer [dom-attr]]
            [clojure.string :as string])
  (:require-macros [lt.macros :refer [behavior defui]]))

(def styles (object/create ::styles
                           :theme "default"))

(behavior ::font-settings
          :desc "App: Font settings"
          :params [{:label "Font family"
                    :type :string}
                   {:label "Size (pt)"
                    :type :number}
                   {:label "Line height (em)"
                    :type :number}]
          :type :user
          :exclusive true
          :triggers #{:object.instant}
          :reaction (fn [this family size line-height]
                      (let [final {:font-family family}
                            final (if size
                                    (assoc final :font-size size)
                                    final)
                            final (if (and line-height
                                           (> line-height 0))
                                    (assoc final :line-height line-height)
                                    final)]
                        (object/merge! styles {:font-settings final}))))
