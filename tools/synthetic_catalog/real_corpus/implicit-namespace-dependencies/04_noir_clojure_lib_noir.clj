(ns noir.util.middleware
  (:use [noir.request :only [*request*]]
        [noir.response :only [redirect]]
        [compojure.core :only [routes]]
        [ring.middleware.defaults :refer [site-defaults wrap-defaults]]
        [hiccup.middleware :only [wrap-base-url]]
        [noir.validation :only [wrap-noir-validation]]
        [noir.cookies :only [wrap-noir-cookies]]
        [noir.session :only [clear! mem wrap-noir-session wrap-noir-flash]]
        [ring.middleware.session.memory :only [memory-store]]
        [ring.middleware.format :refer [wrap-restful-format]])
  (:require [clojure.string :as s])
  (:import java.util.Date))
