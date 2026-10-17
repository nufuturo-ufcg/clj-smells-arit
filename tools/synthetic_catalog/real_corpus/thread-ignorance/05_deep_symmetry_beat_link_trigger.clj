(ns beat-link-trigger.expression-report
  "Generates and supports the report of show expressions."
  (:require [beat-link-trigger.expressions :as expressions]
            [beat-link-trigger.show :as show]
            [beat-link-trigger.show-cues :as cues]
            [beat-link-trigger.show-phrases :as phrases]
            [beat-link-trigger.show-util :as su]
            [beat-link-trigger.util :as util]
            [cheshire.core :as json]
            [clojure.java.io :as io]
            [clojure.string :as str]
            [compojure.route :as route]
            [hiccup.core :as hiccup]
            [hiccup.page :as page]
            [hiccup.util]
            [seesaw.core :as seesaw]
            [taoensso.timbre :as timbre])
  (:import [java.text SimpleDateFormat]
           [java.util Date UUID]))

(defn- expression-section
  "Builds a section of the expressions report if `expressions` is not
  empty. If a subtitle is desired, `title` can be passed as a vector
  of `[title subtitle]`."
  [title id button-code button-title expressions]
  (let [[title subtitle] (if (vector? title) title [title])]
    (when (seq expressions)
      [:div
       [:h2.title.is-4.mt-2.mb-0 {:id id} title
        (when button-code
          [:a.button.is-small.is-link.ml-5 {:href  (str "javascript:" button-code)
                                            :title button-title}
           [:img {:src   "/resources/cog.svg"
                  :width 15}]])]
       (when subtitle
         [:h3.subtitle.is-6.mt-0.mb-0 subtitle])
       [:table.table
        [:thead
         [:tr [:td "Expression"] [:td {:colspan 2} "Actions"] [:td "Value"]]]
        [:tbody
         expressions]]])))

(defn- comment-or-untitled
  "Returns the supplied comment, unless that is blank, in which case
  returns \"Untitled\"."
  [comment]
  (if (str/blank? comment) "Untitled" comment))

(defn cue-expression-disabled-warning
  "Builds a warning message if the expression is not currently in use by
  the cue because the message for that event is set to something other
  than Custom."
  [cue kind]
  (case kind
    (:entered :exited)
    (let [message (get-in cue [:events :entered :message])]
      (when (not= message "Custom")
        [:span.has-text-danger [:br] "Inactive: Enabled Message is &ldquo;" message "&rdquo;"]))

    :started-on-beat
    (let [message (get-in cue [:events :started-on-beat :message])]
      (when (not= message "Custom")
        [:span.has-text-danger [:br] "Inactive: On-Beat Message is &ldquo;" message "&rdquo;"]))

    :started-late
    (let [late-message (get-in cue [:events :started-late :message])
          on-beat-message (get-in cue [:events :started-on-beat :message])]
      (if (= late-message "Same")
        (when (not= on-beat-message "Custom")
          [:span.has-text-danger [:br] "Inactive: On-Beat Message is &ldquo;" on-beat-message "&rdquo;"])
        (when (not= late-message "Custom")
          [:span.has-text-danger [:br] "Inactive: Late Message is &ldquo;" late-message "&rdquo;"])))

    :ended
    (let [late-message (get-in cue [:events :started-late :message])
          on-beat-message (get-in cue [:events :started-on-beat :message])
          late-message (if (= late-message "Same") on-beat-message late-message)]
      (when-not ((set [on-beat-message late-message]) "Custom")
        [:span.has-text-danger [:br] "Inactive: Neither On-Beat or Late is &ldquo;Custom&rdquo;"]))

    nil))

(defn- describe-phrase-cue-expression
  [uuid cue editors kind]
  (let [value (get-in cue [:expressions kind])]
    (when-not (str/blank? value)
      [:tr
       [:td [:div.tooltip (get-in editors [kind :title]) [:span.tooltiptext (get-in editors [kind :tip])]]
        (cue-expression-disabled-warning cue kind)]
       [:td [:a.button.is-small.is-link {:href  (str "javascript:simulatePhraseCueExpression('" uuid "','"
                                                     (:uuid cue) "','" (name kind) "');")
                                         :title "Simulate"}
             [:img {:src   "/resources/play-solid.svg"
                    :width 12}]]]
       [:td [:a.button.is-small.is-link {:href  (str "javascript:editPhraseCueExpression('" uuid "','"
                                                     (:uuid cue) "','" (name kind) "');")
                                         :title "Edit"}
             [:img {:src   "/resources/pen-solid.svg"
                    :width 12}]]]
       [:td [:pre.code.expression [:code.expression.language-clojure value]]]])))

(defn- phrase-cue-expressions
  "Builds the report of expressions for a particular phrase trigger cue."
  [uuid phrase cue]
  (let [editors   @@(requiring-resolve 'beat-link-trigger.editors/show-cue-editors)]
    (expression-section
     [(str "Cue &ldquo;" (comment-or-untitled (:comment cue)) "&rdquo; in Phrase Trigger &ldquo;"
           (comment-or-untitled (:comment phrase)) "&rdquo;")
      (cues/beat-range-for-tooltip (:start cue) (:end cue) phrase)]
     (str "phrase-" uuid "-cue-" (:uuid cue))
     (str "editPhraseCueExpression('" uuid "','" (:uuid cue) "');") "Scroll Cues Editor to this Cue"
     (filter identity (map (partial describe-phrase-cue-expression uuid cue editors)
                           (keys editors))))))
