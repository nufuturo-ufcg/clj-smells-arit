(ns jetty8.servlet
  (:require [clojure.java.browse])
  (:import (java.io            PrintWriter)
           (javax.servlet      ServletConfig)
           (javax.servlet.http HttpServletRequest HttpServletResponse))
  (:gen-class :extends javax.servlet.http.HttpServlet))

(def template (slurp "../index.html"))
