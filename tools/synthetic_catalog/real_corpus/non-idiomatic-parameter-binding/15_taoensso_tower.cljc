(ns taoensso.tower
  "Simple internationalization (i18n) and localization (L10n) library for
  Clojure/Script. Wraps standard Java/Script facilities when possible."
  {:author "Peter Taoussanis, Janne Asmala"}
  #?(:clj
     (:require
      [clojure.string       :as str]
      [clojure.java.io      :as io]
      [taoensso.encore      :as enc]
      [taoensso.timbre      :as timbre]
      [taoensso.tower.utils :as utils :refer [defmem- defmem-*]]))

  #?(:clj (:import
           [java.util Date Locale TimeZone Formatter]
           [java.text Collator NumberFormat DateFormat]))

  #?(:cljs (:require-macros [taoensso.tower  :as tower-macros]))
  #?(:cljs
     (:require
      [clojure.string  :as str]
      [goog.string     :as gstr]
      [goog.string.format]
      [taoensso.tower.utils :as utils]
      [taoensso.encore :as enc :refer-macros []])))

#?(:clj
   (defn parse
     "Parses date/number string as a Date/Number. See `fmt` for possible `style`s
     (default :number)."
     [loc s & [style]]
     (let [loc (jvm-locale loc)
           [type st1 st2] (parse-style style)]
       (case (or type :number)
         :number   (.parse (f-number   loc) s)
         :integer  (.parse (f-integer  loc) s)
         :percent  (.parse (f-percent  loc) s)
         :currency (.parse (f-currency loc) s)

         :date     (.parse (f-date loc st1)     s)
         :time     (.parse (f-time loc st1)     s)
         :dt       (.parse (f-dt   loc st1 st2) s)
         (throw (ex-info (str "Unknown style: " style)
                  {:style style}))))))
