(ns ^:no-doc rewrite-clj.node.coercer
  "Notes:
   - Coercion of records handled specially due different behaviors on different JVMs.
     Must be in equality of specificity of in our protocol extensions.

     From https://clojure.org/reference/protocols:

       if one interface is derived from the other, the more derived is used,
       else which one is used is unspecified"
  (:require
   #?@(:clj
       [[rewrite-clj.node.comment]
        [rewrite-clj.node.fn]
        [rewrite-clj.node.forms]
        [rewrite-clj.node.integer]
        [rewrite-clj.node.keyword :refer [keyword-node]]
        [rewrite-clj.node.meta :refer [meta-node]]
        [rewrite-clj.node.namespaced-map]
        [rewrite-clj.node.protocols :as node :refer [NodeCoerceable coerce]]
        [rewrite-clj.node.quote]
        [rewrite-clj.node.reader-macro :refer [reader-macro-node var-node]]
        [rewrite-clj.node.regex :refer [regex-node pattern-string-for-regex]]
        [rewrite-clj.node.seq :refer [vector-node list-node set-node map-node]]
        [rewrite-clj.node.stringz :as stringz]
        [rewrite-clj.node.token :refer [token-node]]
        [rewrite-clj.node.uneval]
        [rewrite-clj.node.whitespace :as ws]
        [rewrite-clj.parser.impl :as pimpl]]
       :cljs
       [[clojure.string :as string]
        [rewrite-clj.node.comment :refer [CommentNode]]
        [rewrite-clj.node.fn :refer [FnNode]]
        [rewrite-clj.node.forms :refer [FormsNode]]
        [rewrite-clj.node.integer :refer [IntNode]]
        [rewrite-clj.node.keyword :refer [KeywordNode keyword-node]]
        [rewrite-clj.node.meta :refer [MetaNode meta-node]]
        [rewrite-clj.node.namespaced-map :refer [NamespacedMapNode MapQualifierNode]]
        [rewrite-clj.node.protocols :as node :refer [NodeCoerceable coerce]]
        [rewrite-clj.node.quote :refer [QuoteNode]]
        [rewrite-clj.node.reader-macro :refer [ReaderNode ReaderMacroNode DerefNode reader-macro-node var-node]]
        [rewrite-clj.node.regex :refer [RegexNode regex-node pattern-string-for-regex]]
        [rewrite-clj.node.seq :refer [SeqNode vector-node list-node set-node map-node]]
        [rewrite-clj.node.stringz :as stringz :refer [StringNode]]
        [rewrite-clj.node.token :refer [TokenNode SymbolNode token-node]]
        [rewrite-clj.node.uneval :refer [UnevalNode]]
        [rewrite-clj.node.whitespace :refer [WhitespaceNode CommaNode NewlineNode] :as ws]
        [rewrite-clj.parser.impl :as pimpl]])
   [rewrite-clj.reader :as reader])
  #?(:clj
     (:import [rewrite_clj.node.comment CommentNode]
              [rewrite_clj.node.fn FnNode]
              [rewrite_clj.node.forms FormsNode]
              [rewrite_clj.node.integer IntNode]
              [rewrite_clj.node.keyword KeywordNode]
              [rewrite_clj.node.meta MetaNode]
              [rewrite_clj.node.namespaced_map NamespacedMapNode MapQualifierNode]
              [rewrite_clj.node.quote QuoteNode]
              [rewrite_clj.node.reader_macro ReaderNode ReaderMacroNode DerefNode]
              [rewrite_clj.node.regex RegexNode]
              [rewrite_clj.node.seq SeqNode]
              [rewrite_clj.node.stringz StringNode]
              [rewrite_clj.node.token TokenNode SymbolNode]
              [rewrite_clj.node.uneval UnevalNode]
              [rewrite_clj.node.whitespace WhitespaceNode CommaNode NewlineNode])))
