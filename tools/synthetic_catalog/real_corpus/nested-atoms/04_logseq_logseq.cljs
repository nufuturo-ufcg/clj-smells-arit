(ns frontend.state
  "Provides main application state and subscriptions."
  (:require [cljs-bean.core :as bean]
            [cljs.core.async :as async :refer [>!]]
            [clojure.set :as set]
            [clojure.string :as string]
            [datascript.core :as d]
            [dommy.core :as dom]
            [electron.ipc :as ipc]
            [electron.locale :as electron-locale]
            [frontend.db.conn-state :as db-conn-state]
            [frontend.dicts :as dicts]
            [frontend.flows :as flows]
            [frontend.graph-tab :as graph-tab]
            [frontend.mobile.util :as mobile-util]
            [frontend.spec.storage :as storage-spec]
            [frontend.storage :as storage]
            [frontend.util :as util]
            [frontend.util.cursor :as cursor]
            [goog.dom :as gdom]
            [goog.object :as gobj]
            [logseq.common.config :as common-config]
            [logseq.db :as ldb]
            [logseq.db.common.entity-plus :as entity-plus]
            [logseq.shui.dialog.core :as shui-dialog]
            [logseq.shui.hooks :as hooks]
            [logseq.shui.ui :as shui]
            [missionary.core :as m]
            [promesa.core :as p]))

(def ^:private supported-locale-tags
  (into {}
        (map (fn [locale]
               [(string/lower-case (name locale)) locale])
             (keys dicts/dicts))))

(defn- canonical-preferred-language
  "Convert browser/storage locale tags into one supported locale string."
  [language]
  (let [tag (cond
              (keyword? language) (name language)
              (string? language) language
              :else nil)]
    (when-not (string/blank? tag)
      (let [normalized-tag (some-> tag string/trim string/lower-case)
            base-tag (some-> normalized-tag (string/split #"-") first)]
        (some-> (or (get supported-locale-tags normalized-tag)
                    (get supported-locale-tags base-tag)
                    :en)
                name)))))

(def ^:private export-block-text-indent-styles #{"dashes" "spaces" "no-indent"})

(def ^:private legacy-export-block-text-indent-styles
  {"indent-style-dashes" "dashes"
   "indent-style-spaces" "spaces"
   "indent-style-none" "no-indent"})

(defn- normalize-export-block-text-indent-style
  [v]
  (cond
    (nil? v) nil
    (contains? export-block-text-indent-styles v) v
    (contains? legacy-export-block-text-indent-styles v) (get legacy-export-block-text-indent-styles v)
    :else (throw (ex-info "Invalid export block text indent style" {:value v}))))

(defonce ^:large-vars/data-var state
  (let [document-mode? (or (storage/get :document/mode?) false)
        current-graph  (let [url-graph (:graph (util/parse-params))
                             graph (or url-graph
                                       (graph-tab/get-tab-repo)
                                       (storage/get :git/current-repo))]
                         (when graph (ipc/ipc "setCurrentGraph" graph))
                         graph)]
    (atom
     {:client-id                             (str (random-uuid))
      :route-match                           nil
      :today                                 nil
      :system/events                         (async/chan 1000)
      :reactive/custom-queries               (async/chan 1000)
      :notification/show?                    false
      :notification/content                  nil
      :instrument/disabled?                  (storage/get "instrument-disabled")
      ;; TODO: how to detect the network reliably?
      ;; NOTE: prefer to use flows/network-online-event-flow
      :network/online?         true
      :me                      nil
      :git/current-repo        current-graph
      :db/restoring?           nil

      :search/q                              ""
      :search/mode                           nil
      :search/args                           nil
      :search/result                         nil
      :search/engines                        {}
      :search/index-build                    {:running? false
                                              :repo nil
                                              :progress 0
                                              :processed 0
                                              :total 0}

      ;; modals
      :modal/dropdowns                       {}
      :modal/id                              nil

      ;; ui
      :ui/viewport                           {}
      :ui/show-property-dialog?              (atom false)

      ;; left sidebar
      :ui/navigation-item-collapsed?         {}
      :ui/recent-pages                       (or (storage/get :ui/recent-pages) {})

      ;; right sidebar
      :ui/handbooks-open?                    false
      :ui/help-open?                         false
      :ui/fullscreen?                        false
      :ui/settings-open?                     false
      :ui/sidebar-open?                      false
      :ui/sidebar-width                      "40%"
      :ui/left-sidebar-open?                 (boolean (storage/get :ls-left-sidebar-open?))
      :ui/theme                              (or (storage/get :ui/theme) "light")
      :ui/system-theme?                      ((fnil identity (or util/mac? util/win32? false)) (storage/get :ui/system-theme?))
      :ui/custom-theme                       (or (storage/get :ui/custom-theme) {:light {:mode "light"} :dark {:mode "dark"}})
      :ui/wide-mode?                         (storage/get :ui/wide-mode)
      :ui/radix-color                        (storage/get :ui/radix-color)
      :ui/editor-font                        (storage/get :ui/editor-font)

      ;; ui/collapsed-blocks is to separate the collapse/expand state from db for:
      ;; 1. right sidebar
      ;; 2. zoom-in view
      ;; 3. queries
      ;; 4. references
      ;; graph => {container-id {:block-id bool}}
      :ui/collapsed-blocks                   {}
      :ui/sidebar-collapsed-blocks           {}
      :ui/file-component                     nil
      :ui/developer-mode?                    (or (= (storage/get "developer-mode") "true")
                                                 false)
      ;; remember scroll positions of visited paths
      :ui/paths-scroll-positions             (atom {})
      :ui/main-container-scroll-top          (atom nil)
      :ui/shortcut-tooltip?                  (if (false? (storage/get :ui/shortcut-tooltip?))
                                               false
                                               true)
      :ui/scrolling?                         (atom false)
      :ui/show-empty-and-hidden-properties?  (atom {:mode :global
                                                    :show? false})
      :document/mode?                        document-mode?

      :config                                {}
      :block/component-editing-mode?         false
      :editor/op                             (atom nil)
      :editor/start-pos                      (atom nil)
      :editor/pending-new-block              (atom nil)
      :editor/hidden-editors                 #{} ;; page names

      :editor/action                         (atom nil)
      :editor/action-data                    nil
      ;; With label or other data
      :editor/last-saved-cursor              (atom {})
      :editor/editing?                       (atom nil)
      :editor/in-composition?                false
      :editor/content                        (atom {})
      :editor/block                          (atom nil)
      :editor/set-timestamp-block            (atom nil) ;; click rendered block timestamp-cp to set timestamp
      :editor/last-input-time                (atom {})
      :editor/document-mode?                 document-mode?
      :editor/args                           (atom nil)
      :editor/on-paste?                      (atom false)
      :editor/last-key-code                  (atom nil)
      :ui/global-last-key-code               (atom nil)
      :editor/block-op-type                  nil ;; :cut, :copy
      :editor/block-refs                     (atom #{})

      ;; Stores deleted refed blocks, indexed by repo
      :editor/last-replace-ref-content-tx    nil

      :editor/code-block-context             nil
      :editor/latest-shortcut                (atom nil)

      :history/paused?                       (atom false)
      :editor/cursor-range                   (atom nil)
      :editor/container-id                   (atom nil)
      :editor/next-edit-block                (atom nil)
      :editor/raw-mode-block                 (atom nil)
      :editor/virtualized-scroll-fn          nil
      :editor/edit-block-fn                  (atom nil)

      ;; Warning: blocks order is determined when setting this attribute
      :selection/blocks                      (atom [])
      :selection/start-block                 (atom nil)
      ;; nil, :up or :down
      ;; used to determine selection direction when two or more blocks are selected
      :selection/direction                   (atom nil)
      :selection/selected-all?               (atom false)
      :custom-context-menu/show?             false
      :custom-context-menu/links             nil
      :custom-context-menu/position          nil

      ;; pages or blocks in the right sidebar
      ;; It is a list of `[repo db-id block-type block-data]` 4-tuple
      :sidebar/blocks                        '()

      :preferred-language                    (canonical-preferred-language (storage/get :preferred-language))

      ;; electron
      :electron/auto-updater-downloaded      false
      :electron/updater-pending?             false
      :electron/updater                      {}
      :electron/app-base-info                nil
      :electron/user-cfgs                    nil
      :electron/server                       nil
      :electron/window-maximized?            false
      :electron/window-fullscreen?           false

      ;; assets
      :assets/alias-enabled?                 (or (storage/get :assets/alias-enabled?) false)
      :assets/alias-dirs                     (or (storage/get :assets/alias-dirs) [])
      :assets/asset-file-write-finish        (atom {})

      ;; mobile
      :mobile/container-urls                 nil
      :mobile/show-action-bar?               false

      ;; plugin
      :plugin/enabled                        (and util/plugin-platform?
                                                  ;; true false :theme-only
                                                  ((fnil identity true) (storage/get ::storage-spec/lsp-core-enabled)))
      :plugin/preferences                    nil
      :plugin/indicator-text                 nil
      :plugin/installed-plugins              {}
      :plugin/installed-themes               []
      :plugin/installed-slash-commands       {}
      :plugin/installed-ui-items             {}
      :plugin/installed-resources            {}
      :plugin/installed-hooks                {}
      :plugin/installed-services             {}
      :plugin/simple-commands                {}
      :plugin/selected-theme                 nil
      :plugin/selected-unpacked-pkg          nil
      :plugin/marketplace-pkgs               nil
      :plugin/marketplace-stats              nil
      :plugin/installing                     nil
      :plugin/active-readme                  nil
      :plugin/updates-auto-checking?         false
      :plugin/updates-pending                {}
      :plugin/updates-coming                 {}
      :plugin/updates-downloading?           false
      :plugin/updates-unchecked              #{}
      :plugin/navs-settings?                 true
      :plugin/focused-settings               nil ;; plugin id

      ;; pdf
      :pdf/system-win?                       false
      :pdf/current                           nil
      :pdf/ref-highlight                     nil
      :pdf/block-highlight-colored?          (or (storage/get "ls-pdf-hl-block-is-colored") true)
      :pdf/auto-open-ctx-menu?               (not= false (storage/get "ls-pdf-auto-open-ctx-menu"))

      ;; all notification contents as k-v pairs
      :notification/contents                 {}

      :copy/export-block-text-indent-style   (or (normalize-export-block-text-indent-style
                                                  (storage/get :copy/export-block-text-indent-style))
                                                 "dashes")
      :copy/export-block-text-remove-options (or (storage/get :copy/export-block-text-remove-options)
                                                 #{})
      :copy/export-block-text-other-options  (or (storage/get :copy/export-block-text-other-options)
                                                 {})
      :date-picker/date                      nil

      :youtube/players                       {}

      ;; command palette
      :command-palette/commands              (atom [])

      :view/components                       {}
      :view/selected-blocks                  nil

      :srs/mode?                             false

      :srs/cards-due-count                   nil

      :reactive/query-dbs                    {}

      ;; login, userinfo, token, ...
      :auth/refresh-token                    (some-> (storage/get "refresh-token") str)
      :auth/access-token                     nil
      :auth/id-token                         nil

      ;; graph-uuid -> ...
      :rtc/state                             (atom {})
      :rtc/loading-graphs?                   nil
      ;; only latest rtc-log stored here, when a log stream is needed,
      ;; use missionary to create a rtc-log-flow, use (missionary.core/watch <atom>)
      :rtc/log                               (atom nil)
      :rtc/uploading?                        false
      :rtc/downloading-graph-uuid            nil
      :rtc/graphs                            []
      :rtc/online-info                       (atom {})
      :rtc/asset-upload-download-progress    (atom {})
      :rtc/users-info                        (atom {})
      :sync/block-conflicts                  (atom {})

      :user/info                             {:UserGroups (storage/get :user-groups)}
      :encryption/graph-parsing?             false

      :ui/loading?                           {}
      :ui/container-id                       (atom 0)
      :ui/cached-key->container-id           (atom {})
      :feature/enable-sync?                  (storage/get :logseq-sync-enabled)

      :ui/find-in-page                       nil
      :graph/importing                       nil
      :graph/importing-state                 {}
      :graph/loading?                        nil
      :handbook/route-chan                   (async/chan (async/sliding-buffer 1))

      :system/info                           {}
      ;; Whether block is selected
      :ui/select-query-cache                 (atom {})
      :ui/toggle-highlight-recent-blocks?    (atom false)
      :ui/highlight-recent-days              (atom (or (storage/get :ui/highlight-recent-days)
                                                       3))
      :favorites/updated?                    (atom 0)
      :db/async-queries                      (atom {})
      :db/latest-transacted-entity-uuids     (atom {})})))

(deftype PathCursor [ref path meta]
  IAtom

  IMeta
  (-meta [_] meta)

  IEquiv
  (-equiv [this other]
    (identical? this other))

  IDeref
  (-deref [_]
    (get-in (-deref ref) path))

  IWatchable
  (-add-watch [this key callback]
    (add-watch ref (list this key)
               (fn [_ _ oldv newv]
                 (let [old (get-in oldv path)
                       new (get-in newv path)]
                   (when (not= old new)
                     (callback key this old new)))))
    this)

  (-remove-watch [this key]
    (remove-watch ref (list this key))
    this)

  (-notify-watches [_ _ _])

  IHash
  (-hash [this]
    (goog/getUid this))

  IReset
  (-reset! [_ newv]
    (swap! ref assoc-in path newv)
    newv)

  ISwap
  (-swap! [this f]
    (-reset! this (f (-deref this))))
  (-swap! [this f a]
    (-reset! this (f (-deref this) a)))
  (-swap! [this f a b]
    (-reset! this (f (-deref this) a b)))
  (-swap! [this f a b xs]
    (-reset! this (apply f (-deref this) a b xs)))

  IPrintWithWriter
  (-pr-writer [this writer _opts]
    (-write writer "#object [frontend.state.PathCursor ")
    (-write writer (pr-str {:val (-deref this)}))
    (-write writer "]")))

(defn- path-cursor-in
  [ref path]
  (if (instance? PathCursor ref)
    (PathCursor. (.-ref ref) (into (.-path ref) path) nil)
    (PathCursor. ref path nil)))

(defn- path-cursor
  [ref key]
  (path-cursor-in ref [key]))

(defn get-state
  "Returns a plain state value.

  Use this outside component rendering or when the caller does not need reactive
  rendering. Components that render from state should use `use-sub`."
  [ks & {:keys [path-in-sub-atom]}]
  (let [ks-coll?               (coll? ks)
        get-fn                 (if ks-coll? get-in get)
        s                      (get-fn @state ks)
        s-atom?                (util/atom? s)
        path-coll?-in-sub-atom (coll? path-in-sub-atom)]
    (cond
      (and s-atom? path-in-sub-atom path-coll?-in-sub-atom)
      (get-in @s path-in-sub-atom)

      (and s-atom? path-in-sub-atom)
      (get @s path-in-sub-atom)

      s-atom?  @s
      :else    s)))

(defn use-sub
  "Returns a state subscription value for function components."
  [ks & {:keys [path-in-sub-atom]}]
  (let [ks-coll?               (coll? ks)
        get-fn                 (if ks-coll? get-in get)
        s                      (get-fn @state ks)
        s-atom?                (util/atom? s)
        path-coll?-in-sub-atom (coll? path-in-sub-atom)]
    (cond
      (and s-atom? path-in-sub-atom path-coll?-in-sub-atom)
      (first (hooks/use-atom (path-cursor-in s path-in-sub-atom)))

      (and s-atom? path-in-sub-atom)
      (first (hooks/use-atom (path-cursor s path-in-sub-atom)))

      s-atom?  (first (hooks/use-atom s))
      ks-coll? (first (hooks/use-atom (path-cursor-in state ks)))
      :else    (first (hooks/use-atom (path-cursor state ks))))))

(defn user-groups
  []
  (set (get-state [:user/info :UserGroups])))

(defn editing?
  []
  (seq @(:editor/editing? @state)))

(defn developer-mode?
  []
  (:ui/developer-mode? @state))

(defn document-mode?
  []
  (get @state :document/mode?))
