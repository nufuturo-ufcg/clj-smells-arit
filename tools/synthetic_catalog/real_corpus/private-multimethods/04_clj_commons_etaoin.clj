(ns etaoin.api
  "A wrapper over the [W3C WebDriver Specification](https://www.w3.org/TR/webdriver/) to automate popular browsers.

  Tries to normalize differences across the various implementations.

  See the [User Guide](/doc/01-user-guide.adoc) for details and examples.

  This is a rich API:

  **WebDriver**
  - [[with-driver]] [[boot-driver]] [[defaults]] [[defaults-global]] [[when-not-drivers]]
  - [[with-chrome]] [[with-chrome-headless]] [[chrome]] [[chrome-headless]] [[chrome?]] [[when-chrome]] [[when-not-chrome]]
  - [[with-edge]] [[with-edge-headless]] [[edge]] [[edge-headless]] [[when-edge]] [[when-not-edge]]
  - [[with-firefox]] [[with-firefox-headless]] [[firefox]] [[firefox-headless]] [[firefox?]] [[when-firefox]] [[when-not-firefox]]
  - [[with-safari]] [[safari]] [[safari?]] [[when-safari]] [[when-not-safari]]
  - [[driver?]] [[driver-type]] [[running?]] 
  - [[headless?]] [[when-headless]] [[when-not-headless]]
  - [[disconnect-driver]] [[stop-driver]] [[quit]]

  **WebDriver Lower Level Comms**
  -  [[execute]] [[with-http-error]]

  **Driver Sessions**
  - [[get-status]] [[create-session]] [[delete-session]]

  **Querying/Selecting DOM Elements**
  - [[query]] [[query-all]] [[query-tree]] [[query-from]] [[query-all-from]]
  - [[get-active-element]]
  - [[query-from-shadow-root]] [[query-from-shadow-root-el]] [[query-all-from-shadow-root]] [[query-all-from-shadow-root-el]]
  - [[has-shadow-root?]] [[has-shadow-root-el?]]
  - [[exists?]] [[absent?]]
  - [[displayed?]] [[displayed-el?]] [[enabled?]] [[enabled-el?]] [[disabled?]] [[invisible?]] [[visible?]]
  - [[child]] [[children]]
  - [[get-element-tag]] [[get-element-tag-el]]
  - [[get-element-attr]] [[get-element-attr-el]] [[get-element-attrs]]
  - [[get-element-property]] [[get-element-property-el]] [[get-element-properties]]
  - [[has-class?]] [[has-class-el?]] [[has-no-class?]]
  - [[get-element-css]] [[get-element-css-el]] [[get-element-csss]]
  - [[get-element-shadow-root]] [[get-element-shadow-root-el]]
  - [[get-element-text]] [[get-element-text-el]] [[has-text?]]
  - [[get-element-inner-html]] [[get-element-inner-html-el]]
  - [[get-element-value]] [[get-element-value-el]]
  - [[get-element-rect]] [[get-element-rect-el]] [[get-element-size]] [[get-element-size-el]] [[get-element-location]] [[get-element-location-el]] [[get-element-box]] [[intersects?]]
  - [[use-css]] [[with-css]] [[use-xpath]] [[with-xpath]]

  **Browser Navigation**
  - [[go]] [[get-url]] [[get-hash]] [[set-hash]]
  - [[back]] [[forward]]
  - [[refresh]] [[reload]]

  **Mouse/Pointer**
  - [[click]] [[click-el]] [[click-single]] [[click-multi]]
  - [[left-click-on]] [[middle-click-on]] [[right-click-on]] [[mouse-click-on]]
  - [[double-click]] [[double-click-el]]
  - [[drag-and-drop]]
  - [[touch-tap]]

  **Inputs/Forms**
  - [[fill]] [[fill-active]] [[fill-el]] [[fill-multi]]
  - [[fill-human]] [[fill-human-active]] [[fill-human-el]] [[fill-human-multi]]
  - [[select]] [[selected?]] [[selected-el?]]
  - [[upload-file]] [[remote-file]]
  - [[disabled?]] [[enabled?]]
  - [[clear]] [[clear-el]]
  - [[submit]]

  **Cookies**
  - [[get-cookie]] [[get-cookies]]
  - [[set-cookie]]
  - [[delete-cookie]] [[delete-cookies]]

  **Alerts**
  - [[has-alert?]] [[has-no-alert?]]
  - [[get-alert-text]]
  - [[accept-alert]] [[dismiss-alert]]

  **Scrolling**
  - [[get-scroll]]
  - [[scroll]] [[scroll-by]]
  - [[scroll-bottom]] [[scroll-top]]
  - [[scroll-down]] [[scroll-up]] [[scroll-left]] [[scroll-right]]
  - [[scroll-offset]]
  - [[scroll-query]]

  **Scripting**
  - [[js-execute]] [[js-async]] [[js-localstorage-clear]] [[el->ref]] [[add-script]]

  **Browser Windows**
  - [[get-window-handle]] [[get-window-handles]]
  - [[get-window-rect]] [[set-window-rect]]
  - [[get-window-position]] [[set-window-position]]
  - [[get-window-size]] [[set-window-size]]
  - [[maximize]]
  - [[switch-window]] [[switch-window-next]]
  - [[new-window]] [[close-window]]

  **Frames**
  - [[switch-frame]] [[switch-frame-first]] [[switch-frame-parent]] [[switch-frame-top]] [[with-frame]]

  **Page Info**
  - [[get-source]] [[get-title]]

  **Screenshots**
  - [[screenshot]] [[screenshot-element]] [[with-screenshots]]

  **Print to PDF**
  - [[print-page]]

  **Browser Info**
  - [[supports-logs?]] [[get-log-types]] [[get-logs]]
  - [[get-user-agent]]
  - [[with-postmortem]]

  **Waiting**
  - [[doto-wait]] [[wait]] [[with-wait]]
  - [[*wait-interval*]] [[*wait-timeout*]] [[with-wait-interval]] [[with-wait-timeout]]
  - [[wait-exists]] [[wait-absent]]
  - [[wait-visible]] [[wait-invisible]]
  - [[wait-disabled]] [[wait-enabled]]
  - [[wait-has-alert]]
  - [[wait-has-class]]
  - [[wait-has-text]] [[wait-has-text-everywhere]]
  - [[wait-predicate]]
  - [[wait-running]]

  **Browser Timeouts**
  - [[get-timeouts]] [[set-timeouts]]
  - [[get-implicit-timeout]] [[set-implicit-timeout]]
  - [[get-page-load-timeout]] [[set-page-load-timeout]]
  - [[get-script-timeout]] [[set-script-timeout]] [[with-script-timeout]]

  **WebDriver Actions**
  - [[make-action-input]] [[make-key-input]] [[make-mouse-input]] [[make-pen-input]] [[make-pointer-input]] [[make-touch-input]]
  - [[add-action]]
  - [[add-double-pause]]
  - [[add-key-down]] [[add-key-press]] [[add-key-up]] [[with-key-down]]
  - [[add-pause]]
  - [[add-pointer-click]] [[add-pointer-click-el]] [[add-pointer-double-click]] [[add-pointer-double-click-el]]
  - [[add-pointer-down]] [[add-pointer-up]]
  - [[with-pointer-btn-down]] [[with-pointer-left-btn-down]]
  - [[add-pointer-move]] [[add-pointer-move-to-el]]
  - [[add-pointer-cancel]]
  - [[perform-actions]] [[release-actions]]

  **Convenience**
  - [[rand-uuid]]
  "
  (:require
   [babashka.fs :as fs]
   [babashka.process :as p]
   [cheshire.core :as json]
   [clj-commons.slingshot :refer [throw+ try+]]
   [clojure.java.io :as io]
   [clojure.string :as str]
   [clojure.tools.logging :as log]
   [etaoin.impl.client :as client]
   [etaoin.impl.driver :as drv]
   [etaoin.impl.proc :as proc]
   [etaoin.impl.util :as util :refer [defmethods]]
   [etaoin.impl.xpath :as xpath]
   [etaoin.keys :as k]
   [etaoin.query :as query])
  (:import
   (java.text SimpleDateFormat)
   (java.util Base64 Date)))

(defn- dispatch-driver
  "Returns the current driver's type. Used as dispatcher in
  multimethods."
  [driver & _]
  (:type driver))

(defn execute
  "Return response from having `:driver` execute HTTP `:method` request to `:path` with body `:data`.

  Response body automatically converted from JSON to a clojure keywordized map.
  Any HTTP status failure code results in a throw.

  - `:method` HTTP method, e.g. `:get`, `:post`, `:delete`, etc.
  - `:path` a vector of strings/keywords representing a server's path. For example:
      - this: `[:session \"aaaa-bbbb-cccc\" :element \"dddd-eeee\" :find]`
      - becomes: `\"/session/aaaa-bbbb-cccc/element/dddd-eeee/find\"`.
  - `:data` optional body to send. Automatically converted to JSON.

  This can be useful to call when you want to invoke some WebDriver
  implementation specific feature that Etaoin has not otherwise exposed.

  For example, if Etaoin did not already have [[get-url]]:

  ```Clojure
  (def driver (e/firefox))
  (e/go driver \"https://clojure.org\")
  (e/execute {:driver driver
              :method :get
              :path [:session (:session driver) :url]})
  ;; => {:value \"https://clojure.org/\"}
  ```"
  [{:keys [driver method path data]}]
  (client/call driver method path data))

(defn go
  "Have `driver` open `url` in the current browser window.

  Example:

  ```Clojure
  (def ff (firefox))
  (go ff \"http://google.com\")
  ```

  https://www.w3.org/TR/webdriver2/#dfn-navigate-to"
  [driver url]
  (execute {:driver driver
            :method :post
            :path   [:session (:session driver) :url]
            :data   {:url url}}))

(defn get-url
  "Have `driver` return the current url location as a string.

  https://www.w3.org/TR/webdriver2/#dfn-get-current-url"
  [driver]
  (:value (execute {:driver driver
                    :method :get
                    :path   [:session (:session driver) :url]})))

(defn ^:no-doc process-log
  "Remaps some of the log's fields."
  [entry]
  (-> entry
      (update :level (comp keyword str/lower-case))
      (update :source keyword)
      (assoc :datetime (java.util.Date. ^long (:timestamp entry)))))

(defmulti ^:private get-logs*
  {:arglists '([driver logtype])}
  dispatch-driver)

(defmethods get-logs*
  [:chrome :edge]
  [driver logtype]
  (->> (execute {:driver driver
                 :method :post
                 :path   [:session (:session driver) :se :log]
                 :data   {:type logtype}})
       :value
       (mapv process-log)))
