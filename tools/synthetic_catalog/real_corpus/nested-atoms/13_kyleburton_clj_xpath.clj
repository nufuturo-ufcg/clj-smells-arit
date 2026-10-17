(ns clj-xpath.lib
  (:require
   [clojure.string :as str-utils]
   [clj-xpath.util :refer [throwf]]
   [schema.core    :as s])
  (:import
   [java.io             InputStream ByteArrayInputStream]
   [org.w3c.dom         Document Node]
   [javax.xml.parsers   DocumentBuilderFactory]
   [javax.xml.xpath     XPathConstants XPathExpression XPath]
   [javax.xml           XMLConstants]
   [javax.xml.namespace QName]))

(defonce xpath-functions-by-xp-compiler (atom {}))

(defn register-xpath-function [^XPath xp-compiler qualified-name arity f]
  (let [qname     (->qualified-name qualified-name)
        xpfn      (reify javax.xml.xpath.XPathFunction
                    (evaluate [this args]
                      (f this args)))
        fmap-atom (if (-> xpath-functions-by-xp-compiler deref (contains? xp-compiler))
                    (-> xpath-functions-by-xp-compiler deref (get xp-compiler))
                    (do
                      (swap! xpath-functions-by-xp-compiler assoc xp-compiler (atom {}))
                      (-> xpath-functions-by-xp-compiler deref (get xp-compiler))))]
    (swap! fmap-atom
           assoc-in
           [qname arity]
           xpfn)
    (when (nil? (.getXPathFunctionResolver xp-compiler))
      (let [resolver (reify
                       javax.xml.xpath.XPathFunctionResolver
                       (resolveFunction [_this fname arity]
                         (get
                          (->>
                           @fmap-atom
                           (filter
                            (fn [[qname _arities]]
                              (= qname fname)))
                           first
                           second)
                          arity)))]
        (.setXPathFunctionResolver ^XPath xp-compiler resolver)))
    qname))
