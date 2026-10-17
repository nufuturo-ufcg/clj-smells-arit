(ns ^:no-doc zprint.config
  #?(:clj [:refer-clojure :exclude [read-string]])
  #?@(:cljs [[:require-macros
              [zprint.macros :refer
               [dbg dbg-s dbg-pr dbg-s-pr dbg-form dbg-print zfuture]]]])
  (:require
    #?@(:clj [[zprint.macros :refer
               [dbg-pr dbg-s-pr dbg dbg-s dbg-form dbg-print zfuture]]])
    [clojure.data    :as d]
    [clojure.set     :refer [difference]]
    #_clojure.stacktrace
    clojure.string
    [zprint.guide    :refer [areguide defprotocolguide defprotocolguide-s
                             guideguide jrequireguide metaguide odrguide
                             rodguide signatureguide1]]
    [zprint.optionfn :refer [docstring-nl fn*->% meta-base-fn regexfn rodfn
                             rulesfn sort-deps sort-reqs]]
    [zprint.rewrite  :refer [sort-dependencies sort-requires]]
    [zprint.spec     :refer [coerce-to-boolean validate-basic]]
    [zprint.util     :refer [dbg-s-merge dissoc-two]]
    #?@(:bb [[sci.core :as sci]]
        ; To completely remove sci, comment out the following line.
        :clj [[sci.core :as sci]]
        :cljs [[sci.core :as sci]])
    #?(:clj [clojure.edn :refer [read-string]]
       :cljs [cljs.reader :refer [read-string]]))
  #?@(:clj [(:import (java.io InputStreamReader FileReader BufferedReader))]))

(def zfnstyle
  {"->" [:noarg1-body
         {:list {:constant-pair? false},
          :next-inner-restore [[:list :constant-pair?]]}],
   "->>" [:force-nl-body
          {:list {:constant-pair? false},
           :next-inner-restore [[:list :constant-pair?]]}],
   ":import" :force-nl-body,
   ":require" :force-nl-body,
   "=" :hang,
   "alt" :pair-fn,
   "alt!" :pair-fn,
   "alt!!" :pair-fn,
   "and" :hang,
   "apply" :arg1,
   "are" [:guided {:style :areguide}],
   "as->" :arg2,
   "assert-args" :pair-fn,
   "assoc" :arg1-pair,
   "assoc-in" :arg1,
   "binding" :binding,
   "case" :arg1-pair-body,
   "cat" :force-nl,
   "catch" :arg2,
   "comment" :flow-body,
   "cond" :pair-fn,
   "cond-let" :pair-fn,
   "cond->" :arg1-pair-body,
   "cond->>" :arg1-pair-body,
   "condp" :arg2-pair,
   "def" :arg1-body,
   "defc" :arg1-mixin,
   "defcc" :arg1-mixin,
   "defcs" :arg1-mixin,
   "defmacro" :arg1-body,
   "defexpect" [:arg1-body
                {; This replicates :style :respect-nl, but
                 ; is done explicitly here in case that style
                 ; changes, so that the :next-inner-restore will
                 ; still be accurate
                 :list {:respect-nl? true},
                 :map {:respect-nl? true},
                 :vector {:respect-nl? true},
                 :set {:respect-nl? true},
                 :next-inner-restore [[:list :respect-nl?] [:map :respect-nl?]
                                      [:vector :respect-nl?]
                                      [:set :respect-nl?]]}],
   "defmethod" :arg2,
   "defmulti" :arg1-body,
   "defn" :arg1-body,
   "defn-" :arg1-body,
   "defonce" :arg1-body,
   "defproject" [:arg2-pair {:vector {:wrap? false}}],
   "defprotocol" [:guided-body
                  {:style :defprotocolguide,
                   :fn-map {:default-not-none [:none {:style :signature1}]}}],
   "defrecord" :arg2-extend-body,
   "deftest" :arg1-body,
   "deftype" :arg2-extend-body,
   "defui" :arg1-extend,
   "dissoc" [:arg1
             {:list {:constant-pair? false},
              :next-inner-restore [[:list :constant-pair?]]}],
   "do" :none-body,
   "doseq" :binding,
   "dotimes" :binding,
   "doto" :arg1-body,
   "extend" :arg1-extend,
   "extend-protocol" :arg1-extend,
   "extend-type" :arg1-extend,
   "fdef" :arg1-force-nl,
   "filter" :arg1,
   "filterv" :arg1,
   "fn" :fn,
   "fn*" :fn,
   "for" :binding,
   "go-loop" :binding,
   "if" :arg1-body,
   "if-let" :binding,
   "if-not" :arg1-body,
   "if-some" :binding,
   "interpose" :arg1,
   "let" :binding,
   "letfn" [:guided-body
            {:guide [:element :options {:next-inner {:fn-style :fn}}
                     :element-best :options-reset :newline :element-best-*]}],
   "locking" :arg1-body,
   "loop" :binding,
   "map" :arg1,
   "mapcat" :arg1,
   "match" :arg1-pair-body,
   "matchm" :arg1-pair-body,
   "mapv" :arg1,
   "not=" :hang,
   "ns" :arg1-body,
   "or" :hang,
   "proxy" :arg2-fn,
   "reduce" :arg1,
   "reify" :extend,
   "remove" :arg1,
   "reset!" :arg2,
   "s/def" [:arg1-body {:list {:constant-pair-min 2}}],
   "s/fdef" [:arg1-body {:list {:constant-pair-min 2}}],
   "s/and" :gt2-force-nl,
   "s/or" :gt2-force-nl,
   "some->" :force-nl-body,
   "some->>" :force-nl-body,
   "swap!" :arg2,
   "testing" :arg1-body,
   "try" :none-body,
   "when" :arg1-body,
   "when-first" :binding,
   "when-let" :binding,
   "when-not" :arg1-body,
   "when-some" :binding,
   "while" :arg1-body,
   "with-bindings" :arg1-body,
   "with-bindings*" :arg1-body,
   "with-local-vars" :binding,
   "with-meta" :arg1-body,
   "with-open" :binding,
   "with-out-str" :none-body,
   "with-redefs" :binding,
   "with-redefs-fn" :arg1-body,
   :quote [:list
           {:list {:hang? false, :indent 1},
            ; This probably isn't going to make any difference, as
            ; quote? sticks around for a good long time.
            :next-inner-restore [[:list :hang?] [:list :indent]]}]})

(def default-zprint-options
  {:alt? true,
   :agent {:object? false},
   :array {:hex? false, :indent 1, :object? false, :wrap? true},
   :atom {:object? false},
   :binding
     {:flow? false,
      :flow-all-if-any? false,
      :force-nl? false,
      :hang-diff 1,
      :hang-expand 15.0,
      :hang-accept nil,
      :ha-depth-factor 0,
      :ha-width-factor 0,
      :hang? true,
      :indent 2,
      :justify-hang {:hang-expand 15.0},
      :justify-tuning {:binding {:tuning {:hang-flow 4, :hang-flow-limit 30}}},
      :justify {:max-variance 20,
                :no-justify #{"_"},
                :ignore-for-variance nil,
                :max-gap nil,
                :lhs-narrow 2.0,
                :multi-lhs-overlap? true,
                :max-depth 100},
      :justify? false,
      :multi-lhs-hang? false,
      :nl-separator? false,
      :nl-separator-all? false,
      :tuning
        {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false}},
   :cache {:directory ".zprint", :location "HOME"},
   :color? false,
   :color-map {:brace :red,
               :bracket :purple,
               :char :black,
               :comma :none,
               :comment :green,
               :deref :red,
               :false :black,
               :fn :blue,
               :hash-brace :red,
               :hash-paren :green,
               :keyword :magenta,
               :nil :yellow,
               :none :black,
               :number :purple,
               :paren :green,
               :regex :black,
               :symbol :black,
               :syntax-quote-paren :red,
               :quote :red,
               :string :red,
               :true :black,
               :uneval :magenta,
               :user-fn :black,
               :left :none,
               :right :none},
   :comment {:count? false,
             :wrap? true,
             :inline? true,
             :inline-align-style :aligned,
             :border 0,
             :min-space-after-semi 0,
             :smart-wrap? true,
             :smart-wrap {:border 5,
                          ; These regexes will end one comment group and
                          ; start another comment group.
                          ;
                          ; These regexes need two groups at the begining,
                          ; one to capture the semi's and one to capture
                          ; whatever should be considered the spaces.
                          ; These handle things like numbered and
                          ; bulleted lists.
                          :end+start-cg [; Line starts with single letter,
                                         ; but not a or I
                                         #"^(;+)(\s*[b-zA-HJ-Z]\s+)"
                                         ; Line starts with a * or -
                                         #"^(;+)(\s*(?:\*|\-)\s+)"
                                         ; Line starts with single letter
                                         ; followed by .
                                         #"^(;+)(\s*\w\.\s+)"
                                         ; Line starts with one or two
                                         ; digit number followed by
                                         ; period.
                                         #"^(;+)(\s*[0-9]{1,2}\.?\s+)"
                                         ; Line starts with two upper case
                                         ; chars
                                         #"^(;+)(\s*)[A-Z][A-Z]"],
                          ; These regexes will end the previous comment
                          ; group and cause this line to be skipped and
                          ; not included in the next comment group.
                          ;
                          ; These regexes should not have groups.
                          :end+skip-cg [; Blank line
                                        #"^;+\s*$"
                                        ; Line where left paren is first
                                        ; and right paren last
                                        ; character in line.
                                        ;
                                        ; Unneeded given the one below that
                                        ; does a superset of this.
                                        #_#"^;+\s*\(.*\)$"
                                        ; Line containing only capitalized
                                        ; word followed by colon
                                        #"^;+\s*[A-Z]\w+\:$"
                                        ; Line starting with any of ([{ or
                                        ; ending with any of }])
                                        #"^;+\s*(\{|\(|\[|.*\)$|.*\}$|.*\]$)"],
                          ; These regexes match lines which end a comment
                          ; group but also remain in the comment group they
                          ; are ending.
                          ;
                          ; These regexes should not have groups.
                          :end-cg [],
                          :max-variance 30,
                          :space-factor 3,
                          :last-max 5}},
   :configured? false,
   :cwd-zprintrc? false,
   :dbg-ge nil,
   :dbg-print? nil,
   :dbg? nil,
   :dbg-s nil,
   :delay {:object? false},
   :do-in-hang? true,
   :drop? nil,
   :extend {:flow? true,
            :force-nl? true,
            :hang-diff 1,
            :hang-expand 1000.0,
            :hang-accept nil,
            :ha-depth-factor 0,
            :ha-width-factor 0,
            :hang? true,
            :indent 2,
            :modifiers #{"static"},
            :nl-count nil,
            :nl-separator? false,
            :tuning {:hang-flow 1.1,
                     :hang-flow-limit 12,
                     :hang-if-equal-flow? false}},
   :file? false,
   ;  :files {:directory nil :filespec nil}
   :fn-force-nl #{:noarg1-body :noarg1 :force-nl-body :force-nl :flow
                  :arg1-force-nl :arg1-force-nl-body :flow-body
                  :arg2-force-nl-body :arg2-force-nl},
   :fn-gt2-force-nl #{:gt2-force-nl :binding #_:binding-vector :pair-fn},
   :fn-gt3-force-nl #{:gt3-force-nl :arg1-pair :arg1-pair-body},
   :fn-map zfnstyle,
   :fn-type-map {},
   :fn-name nil,
   :fn-obj {:object? false},
   :force-eol-blanks? false,
   :format :on,
   :future {:object? false},
   ; This is used for {:parse {:left-space :keep}}
   :indent 0,
   :input {:range {:start nil,
                   :end nil,
                   :use-previous-!zprint? false,
                   :continue-after-!zprint-error? false}},
   ; When you change :list, you should also change :vector-fn, since it
   ; becomes the current :list when
   :list {:constant-pair-fn nil,
          :constant-pair-min 4,
          :constant-pair? true,
          :force-nl? false,
          :hang-avoid 0.5,
          :hang-diff 1,
          :hang-expand 15.0,
          :hang-accept nil,
          :ha-depth-factor 0,
          :ha-width-factor 0,
          :hang-size 100,
          :hang? true,
          :indent 2,
          :indent-arg nil,
          :indent-only? false,
          :indent-only-style :input-hang,
          :nl-count nil,
          :nl-separator? false,
          :no-wrap-after nil,
          :option-fn nil,
          :pair-hang? true,
          :respect-bl? false,
          :respect-nl? false,
          :replacement-string nil,
          :wrap? false,
          :wrap-coll? true,
          :wrap-after-multi? true,
          :wrap-multi? false,
          :tuning
            {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false},
          :collapse-trailing-right? false,
          :indent-trailing-right? false},
   :map {:indent 2,
         :sort? true,
         :sort-in-code? nil,
         :comma? true,
         :hang? true,
         :hang-expand 1000.0,
         :hang-diff 1,
         ; See zprint_test.clj, def r, (czprint r 28) to see why this was
         ; created and set to 0.  That certainly looks better, but wider
         ; stuff seems better with -1, so for now, we will go with that.
         :hang-adjust -1,
         :hang-accept nil,
         :ha-depth-factor 0,
         :ha-width-factor 0,
         :indent-only? false,
         :key-order nil,
         :key-ignore nil,
         :key-ignore-silent nil,
         :key-color nil,
         :key-depth-color nil,
         :key-value-color nil,
         :key-value-options nil,
         :key-no-sort #{"..."},
         :lift-ns? false,
         :lift-ns-in-code? false,
         :force-nl? nil,
         :nl-separator? false,
         :nl-separator-all? false,
         :flow? false,
         :flow-all-if-any? false,
         :justify? false,
         :justify {:max-variance 20,
                   :ignore-for-variance nil,
                   :no-justify nil,
                   :max-gap nil,
                   :lhs-narrow 2.0,
                   :multi-lhs-overlap? true,
                   :max-depth 100},
         :justify-hang {:hang-expand 1000.0},
         :justify-tuning {:map {:tuning {:hang-flow 4, :hang-flow-limit 30}}},
         :multi-lhs-hang? false,
         :respect-bl? false,
         :respect-nl? false,
         :unlift-ns? false,
         :tuning
           {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false},
         :collapse-trailing-right? false,
         :indent-trailing-right? false},
   :max-depth 1000000,
   :max-depth-string "##",
   :modify-sexpr-by-type nil,
   :parallel? false,
   :max-hang-count 4,
   ; :max-hang-depth used to be 3, but while it helped a bit, there was
   ; at least one bug filed where this caused a problem.  Since we now
   ; support :parallel? true at the repl automatically, we will disable
   ; this performance speedup as less necessary and with unpleasanet
   ; unintended consequences.
   :max-hang-depth 300,
   :max-hang-span 4,
   :max-length 1000000,
   :meta {:split? false},
   :object {:indent 1, :wrap-after-multi? true, :wrap-coll? true},
   :old? true,
   ; This is here to allow various option-fns to communicate as necessary
   :option-fn-map {},
   :output
     {:format :string,
      :focus {:zloc? false, :surround nil},
      :lines nil,
      :elide nil,
      :paragraph
        {:style
           "font-size:20px;font-family: Lucidia Concole, Courier, monospace"},
      :range? nil,
      :real-le? false,
      :real-le-length 20},
   :pair {:flow? false,
          :flow-all-if-any? false,
          :force-nl? nil,
          :hang-diff 1,
          :hang-expand 15.0,
          :hang-accept nil,
          :ha-depth-factor 0,
          :ha-width-factor 0,
          :hang? true,
          :indent 2,
          :justify-hang {:hang-expand 15.0},
          :justify-tuning {:pair {:tuning {:hang-flow 4, :hang-flow-limit 30}}},
          :justify {:max-variance 20,
                    :ignore-for-variance #{":else"},
                    :no-justify nil,
                    :max-gap nil,
                    :lhs-narrow 2.0,
                    :multi-lhs-overlap? true,
                    :max-depth 100},
          :justify? false,
          :multi-lhs-hang? false,
          :nl-separator? false,
          :nl-separator-all? false,
          :tuning
            {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false}},
   :pair-fn {:hang-diff 1,
             :hang-expand 15.0,
             :hang-size 100,
             :hang? true,
             :tuning {:hang-flow 1.1,
                      :hang-flow-limit 12,
                      :hang-if-equal-flow? false}},
   :parse {:interpose nil, :left-space :drop, :ignore-if-parse-fails #{"..."}},
   :parse-string-all? false,
   :parse-string? false,
   :perf-vs-format nil,
   :process-bang-zprint? nil,
   :promise {:object? false},
   :reader-cond {:comma? nil,
                 :force-nl? true,
                 :hang-diff 1,
                 :hang-expand 1000.0,
                 :hang? true,
                 :indent 2,
                 :key-order nil,
                 :sort-in-code? nil,
                 :sort? nil,
                 :tuning {:hang-flow 1.1,
                          :hang-flow-limit 12,
                          :hang-if-equal-flow? false}},
   :record {:hang? true, :record-type? true, :to-string? false},
   ; All of the sets need to be here, so elements can be removed from them
   :remove {:fn-force-nl nil,
            :fn-gt2-force-nl nil,
            :fn-gt3-force-nl nil,
            :extend {:modifiers nil},
            :pair {:justify {:no-justify nil, :ignore-for-variance nil}},
            :binding {:justify {:no-justify nil, :ignore-for-variance nil}},
            :map {:key-no-sort nil,
                  :justify {:no-justify nil, :ignore-for-variance nil}},
            :parse {:ignore-if-parse-fails nil},
            :list {:no-wrap-after nil},
            :set {:no-wrap-after nil},
            :vector-fn {:no-wrap-after nil},
            :vector {:no-wrap-after nil}},
   :return-cvec? false,
   :script {:more-options nil},
   :search-config? false,
   :set {:indent 2,
         :indent-only? false,
         :no-wrap-after nil,
         :respect-bl? false,
         :respect-nl? false,
         :sort? true,
         :sort-in-code? false,
         :wrap-after-multi? true,
         :wrap-coll? true,
         :wrap? true,
         :collapse-trailing-right? false,
         :indent-trailing-right? false,
         :wrap-multi? false},
   :spaces? nil,
   :spec {:docstring? true, :value nil},
   :style nil,
   :style-map
     {:all-hang {:doc "Enable hangs everywhere (which they are by default)",
                 :map {:hang? true},
                 :list {:hang? true},
                 :extend {:hang? true},
                 :pair {:hang? true},
                 :pair-fn {:hang? true},
                 :reader-cond {:hang? true},
                 :record {:hang? true}},
      :anon-fn {:doc "Put anon fn (fn* ...) back to #(... % ...)",
                :fn-map {"fn*" [:none {:list {:option-fn fn*->%}}]}},
      :areguide {:doc "Configurable version of arguide.",
                 :justify? true,
                 :style-fn
                   (fn
                     ([] "argguide-style-fn")
                     ([existing-options new-options style-fn-map style-call]
                      {:list {:option-fn (partial areguide
                                                  (merge-deep style-fn-map
                                                              style-call))}}))},
      :areguidex {:doc "Allow modification of areguide in :fn-map",
                  :list {:option-fn (partial areguide {:justify? true})}},
      :areguide-nj
        {:doc "Do nice are formatting, but don't justify, use only in :fn-map",
         :style-call :areguide,
         :justify? false},
      :backtranslate
        {:doc "Turn quote, deref, var, unquote into reader macros",
         :fn-map
           {"quote" [:replace-w-string {} {:list {:replacement-string "'"}}],
            "clojure.core/deref" [:replace-w-string {}
                                  {:list {:replacement-string "@"}}],
            "var" [:replace-w-string {} {:list {:replacement-string "#'"}}],
            "clojure.core/unquote" [:replace-w-string {}
                                    {:list {:replacement-string "~"}}]}},
      :binding-nl {:doc "Add a blank line after every value that flowed",
                   :binding {:indent 0, :nl-separator? true}},
      :binding-nl-all {:doc "Add a blank line between every pair",
                       :binding {:indent 0, :nl-separator-all? true}},
      :collapse-trailing-right {:doc "collapse-trailing-right for everything",
                                :list {:collapse-trailing-right? true},
                                :map {:collapse-trailing-right? true},
                                :set {:collapse-trailing-right? true},
                                :vector {:collapse-trailing-right? true},
                                :vector-fn {:collapse-trailing-right? true}},
      :community {:doc "Modify defaults to format to 'community' approach",
                  :binding {:indent 0},
                  :fn-map {"apply" :none,
                           "assoc" :none,
                           "filter" :none,
                           "filterv" :none,
                           "map" :none,
                           "mapv" :none,
                           "reduce" :none,
                           "remove" :none,
                           "with-meta" :none-body},
                  :list {:indent-arg 1},
                  :map {:indent 0},
                  :pair {:indent 0}},
      :dark-color-map {:doc
                         "A color map that is pretty good for dark backgrounds",
                       :color-map {:brace :white,
                                   :bracket :white,
                                   :char :bright-cyan,
                                   :comma :bright-white,
                                   :comment :bright-black,
                                   :deref :red,
                                   :false :bright-magenta,
                                   :fn :bright-red,
                                   :hash-brace :white,
                                   :hash-paren :white,
                                   :keyword :bright-blue,
                                   :nil :bright-magenta,
                                   :none :white,
                                   :number :bright-magenta,
                                   :paren :white,
                                   :quote :bright-yellow,
                                   :regex :bright-cyan,
                                   :string :bright-green,
                                   :symbol :bright-white,
                                   :syntax-quote :bright-yellow,
                                   :syntax-quote-paren :white,
                                   :true :bright-magenta,
                                   :uneval :bright-red,
                                   :unquote :bright-yellow,
                                   :unquote-splicing :bright-yellow,
                                   :user-fn :bright-yellow,
                                   :left :bright-white,
                                   :right :bright-white},
                       :uneval {:color-map {:brace :white,
                                            :bracket :white,
                                            :char :bright-cyan,
                                            :comma :bright-white,
                                            :comment :bright-black,
                                            :deref :red,
                                            :false :bright-magenta,
                                            :fn :bright-red,
                                            :hash-brace :white,
                                            :hash-paren :white,
                                            :keyword :bright-blue,
                                            :nil :bright-magenta,
                                            :none :white,
                                            :number :bright-magenta,
                                            :paren :white,
                                            :quote :bright-yellow,
                                            :regex :bright-cyan,
                                            :string :bright-green,
                                            :symbol :bright-white,
                                            :syntax-quote :bright-yellow,
                                            :syntax-quote-paren :white,
                                            :true :bright-magenta,
                                            :uneval :bright-red,
                                            :unquote :bright-yellow,
                                            :unquote-splicing :bright-yellow,
                                            :user-fn :bright-yellow,
                                            :left :bright-white,
                                            :right :bright-white}}},
      :defprotocolguide {:doc "Allow alteration of defprotocol in :fn-map",
                         :list {:option-fn defprotocolguide}},
      :docstring-nl {:doc "Put real \n in docstrings in structures.",
                     :fn-map {"defn" [:arg1-body
                                      {:list {:option-fn docstring-nl}}]}},
      :extend-nl {:doc "Add a blank line between protocols",
                  :extend {:flow? true, :indent 0, :nl-separator? true}},
      :how-to-ns {:doc "Make newlines and indentation match 'how to ns'",
                  :fn-map {"ns" [:arg1-body
                                 {:fn-map {":import" [:flow
                                                      {:list {:hang? true}}],
                                           ":require" :flow},
                                  :list {:hang? false, :indent-arg 1}}]}},
      :guideguide {:doc "output guides themselves, experimental",
                   :vector {:option-fn guideguide}},
      :hiccup {:doc "Format vectors containing hiccup information better",
               :vector
                 {:option-fn
                    (fn
                      ([] "hiccup-option-fn")
                      ([opts n exprs]
                       (let [hiccup? (and (>= n 2)
                                          (or (keyword? (first exprs))
                                              (symbol? (first exprs)))
                                          (map? (second exprs))
                                          ; Disambiguate destructuring maps
                                          (not (some #{:keys :syms :strs :as
                                                       :or}
                                                     (keys (second exprs)))))]
                         (cond (and hiccup? (not (:fn-format (:vector opts))))
                                 {:vector {:fn-format :arg1-force-nl}}
                               (and (not hiccup?) (:fn-format (:vector opts)))
                                 {:vector {:fn-format nil}}
                               :else nil)))),
                  :wrap? false},
               :vector-fn {:indent 1, :indent-arg 1}},
      :indent-only {:doc "Enable indent only for every type of structure",
                    :comment {:wrap? false},
                    :list {:indent-only? true},
                    :map {:indent-only? true},
                    :set {:indent-only? true},
                    :tagged-literal {:indent-only? true},
                    ; Should we also set :vector-fn to :indent-only?  That
                    ; is only used by :fn-format, so it might confuse
                    ; people if we did that.
                    :vector {:indent-only? true}},
      :indent-trailing-right {:doc "indent-trailing-right for everything",
                              :list {:indent-trailing-right? true},
                              :map {:indent-trailing-right? true},
                              :set {:indent-trailing-right? true},
                              :vector {:indent-trailing-right? true},
                              :vector-fn {:indent-trailing-right? true}},
      :justified {:doc "Justify everything possible",
                  :binding {:justify? true},
                  :map {:justify? true},
                  :pair {:justify? true}},
      :justified-original
        {:doc "Justify everything using pre-1.1.2 approach",
         :binding {:justify? true, :justify {:max-variance 1000}},
         :map {:justify? true, :justify {:max-variance 1000}},
         :pair {:justify? true, :justify {:max-variance 1000}}},
      :keyword-respect-nl
        {:doc "When a vector starts with a :keyword, :respect-nl in it",
         :vector {:option-fn-first
                    (fn
                      ([] "keyword-respect-nl-option-fn-first")
                      ([options element]
                       (let [k? (keyword? element)]
                         (when (not= k? (:respect-nl? (:vector options)))
                           {:vector {:respect-nl? k?}}))))}},
      :map-nl {:doc "Add newline after every value that flows",
               :map {:indent 0, :nl-separator? true}},
      :map-nl-all {:doc "Add newline between all map pairs",
                   :map {:indent 0, :nl-separator-all? true}},
      :meta-base {:doc
                    "Alternative format for metadata. Experimental. Deprecated",
                  :list {:option-fn meta-base-fn},
                  :next-inner-restore [[:list :option-fn]]},
      :meta-alt {:doc "Alternative for metadata. Experimental. Deprecated",
                 :fn-map {"def" [:arg2 {:style :meta-base}],
                          "deftest" [:arg1-body {:style :meta-base}]}},
      :meta-guide
        {:doc "Alternative for metadata. Experimental.",
         :one-line-ok? true,
         :style-fn (fn
                     ([] "meta-guide")
                     ([existing-options new-otions style-fn-map style-call]
                      {:fn-map
                         {"def" [:arg2
                                 {:list {:option-fn (partial metaguide
                                                             (merge-deep
                                                               style-fn-map
                                                               style-call))}}],
                          "deftest" [:arg1-body
                                     {:list {:option-fn
                                               (partial metaguide
                                                        (merge-deep
                                                          style-fn-map
                                                          style-call))}}]}}))},
      :minimal-smart-wrap {:doc "Do the minimal smart-wrap",
                           :comment {:smart-wrap {:last-max 80,
                                                  :border 0,
                                                  :max-variance 200,
                                                  :space-factor 100,
                                                  :end-cg [; If it ends with
                                                           ; a period, it
                                                           ; ends the cg.
                                                           #"\.$"]}}},
      :sentence-smart-wrap
        {:doc "Don't run sentences together if they aren't already.",
         :comment {:smart-wrap {:end-cg [; If it ends with
                                         ; a period, it ends
                                         ; the cg.
                                         #"\.$"]}}},
      :moustache {:doc "Format moustache elements nicely",
                  :fn-map {"app" [:flow {:style :vector-pairs}]}},
      :multi-lhs-hang {:doc "Allow multi-lhs-hang in all three places.",
                       :pair {:multi-lhs-hang? true},
                       :binding {:multi-lhs-hang? true},
                       :map {:multi-lhs-hang? true}},
      :vector-pairs {:doc "Consider vectors 'constants' for constant pairing",
                     :list {:constant-pair-min 1,
                            :constant-pair-fn #(or (keyword? %)
                                                   (string? %)
                                                   (number? %)
                                                   (= true %)
                                                   (= false %)
                                                   (vector? %))},
                     :pair {:justify? true},
                     :next-inner {:list {:constant-pair-min 4,
                                         :constant-pair-fn nil},
                                  :pair {:justify? false}}},
      :no-hang {:doc "Turn off hang for every structure",
                :map {:hang? false},
                :list {:hang? false},
                :extend {:hang? false},
                :pair {:hang? false},
                :pair-fn {:hang? false},
                :reader-cond {:hang? false},
                :record {:hang? false}},
      :pair-nl {:doc "Add a blank line after every value that flowed",
                :pair {:indent 0, :nl-separator? true}},
      :pair-nl-all {:doc "Add a blank line between every pair",
                    :pair {:indent 0, :nl-separator-all? true}},
      :fast-hang {:doc "Speed up formatting of very deeply nested structures",
                  :binding {:hang-accept 100, :ha-width-factor -600},
                  :extend {:hang-accept 100, :ha-width-factor -600},
                  :list {:hang-accept 100, :ha-width-factor -300},
                  :map {:hang-accept 0, :ha-depth-factor 15},
                  :pair {:hang-accept 20, :ha-width-factor -150},
                  :vector-fn {:hang-accept 100, :ha-width-factor -300}},
      :odr {:doc "justify columns of rules, experimental",
            :vector {:option-fn odrguide}},
      :quote-wrap {:doc "Wrap quoted lists to right margin, like vectors",
                   :fn-map {:quote [:wrap
                                    {:list {:indent 1},
                                     :next-inner-restore [[:list :indent]]}]}},
      :jrequireguide {:list {:option-fn (partial jrequireguide :require)}},
      :jrequiremacrosguide {:list {:option-fn (partial jrequireguide
                                                       :require-macros)}},
      :jimportguide {:list {:option-fn (partial jrequireguide :import)}},
      :require-justify
        {:doc "Justify namespaces in :require",
         :max-variance 20,
         :style-fn (fn
                     ([] "require-justify")
                     ([existing-options new-otions style-fn-map style-call]
                      {:fn-map {":require" [:flow
                                            {:style :jrequireguide,
                                             :pair
                                               {:justify
                                                  {:max-variance
                                                     (:max-variance
                                                       (merge-deep
                                                         style-fn-map
                                                         style-call))}}}]}}))},
      :require-macros-justify
        {:doc "Justify namespaces in :require-macros",
         :max-variance 20,
         :style-fn (fn
                     ([] "require-macros-justify")
                     ([existing-options new-otions style-fn-map style-call]
                      {:fn-map {":require-macros"
                                  [:flow
                                   {:style :jrequiremacrosguide,
                                    :pair {:justify
                                             {:max-variance
                                                (:max-variance
                                                  (merge-deep
                                                    style-fn-map
                                                    style-call))}}}]}}))},
      :import-justify
        {:doc "Justify :import",
         :max-variance 1000,
         :style-fn (fn
                     ([] "import-justify")
                     ([existing-options new-otions style-fn-map style-call]
                      {:fn-map {":import" [:flow
                                           {:style :jimportguide,
                                            :pair
                                              {:justify
                                                 {:max-variance
                                                    (:max-variance
                                                      (merge-deep
                                                        style-fn-map
                                                        style-call))}}}]}}))},
      :ns-justify
        {:doc "Justify :require, :require-macros, :import in ns",
         :require-macros-max-variance 20,
         :require-max-variance 20,
         :import-max-variance 1000,
         :style-fn (fn
                     ([] "ns-justify")
                     ([existing-options new-otions style-fn-map style-call]
                      (let [merged-options (merge-deep style-fn-map style-call)]
                        {:style [{:style-call :require-justify,
                                  :max-variance (:require-max-variance
                                                  merged-options)}
                                 {:style-call :require-macros-justify,
                                  :max-variance (:require-macros-max-variance
                                                  merged-options)}
                                 {:style-call :import-justify,
                                  :max-variance (:import-max-variance
                                                  merged-options)}]})))},
      :original-tuning
        {:doc "Original tuning prior to stability fixes for multiple passes",
         :binding {:hang-expand 2.0,
                   :justify-hang {:hang-expand 5.0},
                   :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :extend {:tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :list {:hang-expand 2.0,
                :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :map {:tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :pair {:hang-expand 2.0,
                :justify-hang {:hang-expand 5.0},
                :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :pair-fn {:hang-expand 2.0,
                   :hang-size 10,
                   :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}},
         :reader-cond {:tuning {:hang-flow-limit 10,
                                :hang-if-equal-flow? true}},
         :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true},
         :vector-fn {:hang-expand 2.0,
                     :tuning {:hang-flow-limit 10, :hang-if-equal-flow? true}}},
      :regex-example
        {:doc "An example of how to use regex rules to recognize fns",
         :fn-map {:default-not-none
                    [:none
                     {:list {:option-fn (partial regexfn
                                                 [#"^are" {:fn-style "are"}
                                                  #"^when"
                                                  {:fn-style "when"}])}}]}},
      :rules-example
        {:doc "An example of how to use rulesfn to recognize fns",
         :fn-map {:default-not-none
                    [:none
                     {:list {:option-fn (partial
                                          rulesfn
                                          [#(> (count %) 20)
                                           {:guide [:element :newline
                                                    :element-wrap-flow-*]}
                                           #"^are" {:fn-style "are"} #"^when"
                                           {:fn-style "when"}])}}]}},
      :require-pair
        {:doc "Clarify namespaces in :require",
         :fn-map {":require" [:none
                              {:vector {:option-fn
                                          (fn
                                            ([opts n exprs]
                                             (if-not (clojure.string/includes?
                                                       (str (first exprs))
                                                       ".")
                                               {:vector {:fn-format nil}}
                                               {:vector {:fn-format :none},
                                                :vector-fn {:constant-pair-min
                                                              1}}))
                                            ([] "require-pair-option-fn"))}}]}},
      :respect-bl {:doc "Enable respect blank lines for every type",
                   :list {:respect-bl? true},
                   :map {:respect-bl? true},
                   :vector {:respect-bl? true},
                   :set {:respect-bl? true},
                   :tagged-literal {:respect-bl? true}},
      :respect-bl-off {:doc "Disable respect blank lines for every type",
                       :list {:respect-bl? false},
                       :map {:respect-bl? false},
                       :vector {:respect-bl? false},
                       :set {:respect-bl? false},
                       :tagged-literal {:respect-bl? false}},
      :respect-nl {:doc "Enable respect newlines for every type",
                   :list {:respect-nl? true},
                   :map {:respect-nl? true},
                   :vector {:respect-nl? true},
                   :set {:respect-nl? true},
                   :tagged-literal {:respect-nl? true}},
      :respect-nl-off {:doc "Disable respect newline for every type",
                       :list {:respect-nl? false},
                       :map {:respect-nl? false},
                       :vector {:respect-nl? false},
                       :set {:respect-nl? false},
                       :tagged-literal {:respect-nl? false}},
      :rod {:doc "Rules of defn, with newlines between arities.",
            :multi-arity-nl? true,
            :one-line-ok? false,
            :style-fn (fn
                        ([] "rod-style-fn")
                        ([existing-options new-options style-fn-map style-call]
                         {:fn-map {"defn" [:none
                                           {:list {:option-fn
                                                     (partial rodfn
                                                              (merge-deep
                                                                style-fn-map
                                                                style-call))}}],
                                   "defn-" "defn"}}))},
      :rod-no-ma-nl {:doc "Rules of defn, no newlines between arities.",
                     :multi-arity-nl? false,
                     :one-line-ok? false,
                     :style-call :rod},
      :rod-config {:doc "DEPRECATED, here for backward compatibility",
                   :one-line-ok? false,
                   :multi-arity-nl? false,
                   :style-fn
                     (fn
                       ([] "rod-config-style-fn")
                       ([existing-options new-options style-fn-map style-call]
                        {:fn-map {"defn" [:none
                                          {:list {:option-fn
                                                    (partial rodfn
                                                             (merge-deep
                                                               style-fn-map
                                                               style-call))}}],
                                  "defn-" "defn"}}))},
      :signature1 {:doc
                     "defprotocol signatures with doc on newline, experimental",
                   :list {:option-fn signatureguide1}},
      :sort-dependencies {:doc "sort dependencies in lein defproject files",
                          :fn-map {"defproject" [:arg2-pair
                                                 {:vector {:wrap? false},
                                                  :list {:option-fn
                                                           sort-deps}}]}},
      :sort-require
        {:doc "Sort requires & refers in ns macro, possibly with regexes.",
         :regex-vec [],
         :sort-refer? true,
         :style-fn (fn
                     ([] "sort-require-config")
                     ([existing-options new-options style-fn-map style-call]
                      {:fn-map {"ns" [:arg1-body
                                      {:list {:option-fn
                                                (partial
                                                  sort-reqs
                                                  (merge-deep
                                                    style-fn-map
                                                    style-call))}}]}}))}},
   :tab {:expand? true, :size 8},
   :tagged-literal
     {:hang-diff 1,
      :hang-expand 1000.0,
      :hang? true,
      :indent 1,
      :tuning {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false},
      :indent-only? false,
      :respect-bl? false,
      :respect-nl? false},
   :test-for-eol-blanks? false,
   :trim-comments? nil,
   :tuning {; do hang if (< (/ hang-count flow-count) :hang-flow)
            :hang-flow 1.1,
            ; if the :fn-style is hang, then this become the :hang-flow
            ; above
            :hang-type-flow 1.5,
            ; when (> hang-count :hang-flow-limit),
            ;  hang if (<= (dec hang-count) flow-count)
            :hang-flow-limit 12,
            ; this is added to the count of hanging lines before the
            ; comparison when doing the one with :hang-flow or
            ; :hang-type-flow. Note that :map has its own :hang-adjust
            ; which overides this general one.
            :general-hang-adjust -1,
            :hang-if-equal-flow? false},
   :uneval {:color-map {:brace :yellow,
                        :bracket :yellow,
                        :char :magenta,
                        :comma :none,
                        :comment :green,
                        :deref :yellow,
                        :false :yellow,
                        :fn :cyan,
                        :hash-brace :yellow,
                        :hash-paren :yellow,
                        :keyword :yellow,
                        :nil :yellow,
                        :none :yellow,
                        :number :yellow,
                        :paren :yellow,
                        :quote :yellow,
                        :regex :yellow,
                        :string :yellow,
                        :symbol :cyan,
                        :syntax-quote-paren :yellow,
                        :true :yellow,
                        :uneval :magenta,
                        :user-fn :cyan,
                        :left :none,
                        :right :none}},
   :user-fn-map {},
   :vector {:indent 1,
            :binding? false,
            :option-fn-first nil,
            :option-fn nil,
            :fn-format nil,
            :force-nl? false,
            :hang? nil,
            :no-wrap-after nil,
            :respect-bl? false,
            :respect-nl? false,
            :wrap-after-multi? true,
            :wrap-coll? true,
            :wrap? true,
            :wrap-multi? false,
            :indent-only? false,
            :collapse-trailing-right? false,
            :indent-trailing-right? false},
   ; Needs to have same keys as :list, since this replaces :list when
   ; vectors are formatted as functions.
   :vector-fn
     {:constant-pair-fn nil,
      :constant-pair-min 4,
      :constant-pair? true,
      :hang-avoid 0.5,
      :hang-diff 1,
      :hang-expand 15.0,
      :hang-accept nil,
      :ha-depth-factor 0,
      :ha-width-factor 0,
      :hang-size 100,
      :hang? true,
      :indent 2,
      :indent-arg nil,
      :indent-only? false,
      :indent-only-style :input-hang,
      :nl-count nil,
      :no-wrap-after nil,
      :pair-hang? true,
      :respect-bl? false,
      :respect-nl? false,
      :wrap-coll? true,
      :wrap-after-multi? true,
      :wrap-multi? true,
      :tuning {:hang-flow 1.1, :hang-flow-limit 12, :hang-if-equal-flow? false},
      :collapse-trailing-right? false,
      :indent-trailing-right? false},
   :width 80,
   :url {:cache-dir "urlcache", :cache-secs 300},
   :zipper? false,
   :!zprint-elide-skip-next? false})

(defn merge-with-fn
  "Take two arguments of things to merge and figure it out.
  Works for sets too."
  [val-in-result val-in-latter]
  (cond (and (map? val-in-result) (map? val-in-latter))
          (merge-with merge-with-fn val-in-result val-in-latter)
        (and (set? val-in-result) (set? val-in-latter))
          (apply conj val-in-result (seq val-in-latter))
        :else val-in-latter))

(defn merge-deep
  "Do a merge of maps all the way down."
  [& maps]
  (apply merge-with merge-with-fn maps))

(def opts
  {:namespaces {'clojure.core {'jrequireguide zprint.guide/jrequireguide,
                               'defprotocolguide zprint.guide/defprotocolguide,
                               'defprotocolguide-s
                                 zprint.guide/defprotocolguide-s,
                               'signatureguide1 zprint.guide/signatureguide1,
                               'guideguide zprint.guide/guideguide,
                               'rodguide zprint.guide/rodguide,
                               'areguide zprint.guide/areguide,
                               'odrguide zprint.guide/odrguide,
                               'rodfn zprint.optionfn/rodfn,
                               'rulesfn zprint.optionfn/rulesfn,
                               'regexfn zprint.optionfn/regexfn,
                               'sort-deps zprint.optionfn/sort-deps,
                               'sort-reqs zprint.optionfn/sort-reqs,
                               'docstring-nl zprint.optionfn/docstring-nl,
                               'meta-base-fn zprint.optionfn/meta-base-fn,
                               'fn*->% zprint.optionfn/fn*->%,
                               'merge-deep zprint.config/merge-deep,
                               'metaguide zprint.guide/metaguide}}})
