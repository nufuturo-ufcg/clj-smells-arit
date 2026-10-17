(ns athens.views.right-sidebar.shared
  (:require
    [athens.common-db :as common-db]
    [athens.db :as db]
    [athens.parse-renderer :as parse-renderer]
    [athens.reactive :as reactive]
    [athens.router :as router]
    [athens.views.blocks.core :as blocks]
    [athens.views.pages.graph :as graph]
    [athens.views.pages.node-page :as node-page]
    [re-frame.core :as rf]
    [reagent.core :as r]))

(defn get-eid
  [item-block]
  (let [{:keys [type name]} item-block]
    (cond
      (= type "page") [:node/title name]
      (= type "graph") [:node/title name]
      :else [:block/uid name])))

(defn create-sidebar-list
  "Accepts right-sidebar as a map of uids and entities.
  Entity contains either the block uid or node title, and additionally open/close state and whether the page is a graph view or not."
  [items]
  (doall
    (mapv (fn [entity]
            (let [{:keys [open?
                          name
                          source-uid
                          type]}      entity
                  eid                 (get-eid entity)
                  {:keys [node/title
                          block/string
                          block/uid]} (reactive/get-reactive-right-sidebar-item eid)
                  ;; NOTE: add support for BlockTypeProtocol
                  entity-title        (parse-renderer/parse-to-text (or title string))]
              {:isOpen     open?
               :key        source-uid
               :id         source-uid
               :type       type
               :onRemove   #(rf/dispatch [:right-sidebar/close-item source-uid])
               :onToggle   #(rf/dispatch [:right-sidebar/toggle-item source-uid])
               :onNavigate (if (= type "page")
                             #(router/navigate-page title %)
                             #(router/navigate-uid uid %))
               :title      entity-title
               :children   (r/as-element (cond
                                           (= type "graph") [graph/page name]
                                           (= type "page")  [node-page/page eid {:size "sm"}]
                                           :else            [blocks/page eid {:size "sm"}]))}))

          items)))
