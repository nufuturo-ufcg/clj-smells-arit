(ns vrksasana.catalog)

(def *state
  (atom {}))

(defn reset []
  (reset! *state
          {:counter 0
           :ground-name->ground             {}
           :season-name->season             {}
           :season-name->season-attributres {}
           :tree-name->tree                 {}
           :default-ground-name             nil
           :ground->current-season-name     {}
           :tree->seasons                   {}
           :tree->current-season            {}
           :season->trees                   {}}))

(defn ground-name->ground [ground-name]
  (get-in @*state [:ground-name->ground ground-name]))

(defn season-name->season-attributres [season-name]
  (get-in @*state [:season-name->season-attributres season-name]))

(defn tree-name->tree [tree-name]
  (-> @*state :tree-name->tree (get tree-name)))
