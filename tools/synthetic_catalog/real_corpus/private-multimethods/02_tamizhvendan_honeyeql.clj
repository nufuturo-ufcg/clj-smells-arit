(ns honeyeql.core
  (:require [clojure.data.json :as json]
            [clojure.string :as str]
            [edn-query-language.core :as eql]
            [honey.sql.helpers :as hsql-helpers]
            [honeyeql.db-adapter.core :as db]
            [honeyeql.debug :refer [trace>>]]
            [honeyeql.dsl :as dsl]
            [honeyeql.meta-data :as heql-md]
            [inflections.core :as inf]))

(defn- function-attribute-ident? [x]
  (and (vector? x)
       (keyword? (first x))
       (not (qualified-keyword? (first x)))))

(defn- eql-node->attr-ident [{:keys [key type dispatch-key]}]
  (cond
    (and (= :prop type) (keyword? key)) key
    (and (= :prop type) (function-attribute-ident? key)) (second key)
    (and (= :prop type) (dsl/alias-attribute-ident? key)) (if (function-attribute-ident? (first key))
                                                            (second (first key))
                                                            (first key))
    (and (= :join type) dispatch-key) key))

(defn- composite-ref-keys? [{:attr.column.ref/keys [left right]}]
  (and (coll? left) (coll? right) (= (count left) (count right))))

(defn- composite-pred-join [heql-meta-data {:attr.column.ref/keys [left right]} alias pred-fn]
  (->> (map (fn [[left right]]
              (pred-fn heql-meta-data
                       #:attr.column.ref{:left left :right right}
                       alias))
            (partition 2 (interleave left right)))
       (concat [:and])
       vec))

(defn- one-to-*-join-predicate [heql-meta-data {:attr.column.ref/keys [left right] :as ref-keys} alias]
  (if (composite-ref-keys? ref-keys)
    (composite-pred-join heql-meta-data ref-keys alias one-to-*-join-predicate)
    [:=
     (keyword (str (:parent alias) "." (heql-md/attr-column-name heql-meta-data left)))
     (keyword (str (:self alias) "." (heql-md/attr-column-name heql-meta-data right)))]))

(defn- many-to-many-join-predicate [heql-meta-data {:attr.column.ref/keys [left right]
                                                    :as                   join-attr-md} alias assoc-table-alias]
  (let [{:attr.column.ref.associative/keys [left-ident right-ident]} join-attr-md]
    [:and
     [:=
      (keyword (str (:parent alias) "." (heql-md/attr-column-name heql-meta-data left)))
      (keyword (str assoc-table-alias "." (heql-md/attr-column-name heql-meta-data left-ident)))]
     [:=
      (keyword (str assoc-table-alias "." (heql-md/attr-column-name heql-meta-data right-ident)))
      (keyword (str (:self alias) "." (heql-md/attr-column-name heql-meta-data right)))]]))

(defmethod ^{:private true} dsl/eql->hsql :root [db-adapter heql-meta-data eql-node]
  (dsl/eql->hsql db-adapter heql-meta-data (first (:children eql-node))))

(defn- eql-ident->hsql-predicate [db-adapter [attr-ident value] alias]
  (let [heql-meta-data (:heql-meta-data db-adapter)
        attr-col-name  (heql-md/attr-column-name heql-meta-data attr-ident)
        attr-value     (heql-md/coerce-attr-value db-adapter :from-db attr-ident value)]
    [:= (keyword (str (:self alias) "." attr-col-name)) attr-value]))

(defn- eql-ident-key->hsql-predicate [db-adapter eql-ident-key alias]
  (let [predicates (map #(eql-ident->hsql-predicate db-adapter % alias) (partition 2 eql-ident-key))]
    (if (< 1 (count predicates))
      (conj predicates :and)
      (first predicates))))

(defn- resolve-group-by-column [db-adapter eql-node attr-ident]
  (->> (:children eql-node)
       (filter #(= attr-ident (:key %)))
       first
       :alias
       (db/resolve-one-to-one-relationship-alias db-adapter)))

(defn- hsql-column
  ([db-adapter attr-ident-or-rel-attr-ident eql-node]
   (hsql-column db-adapter attr-ident-or-rel-attr-ident eql-node false))
  ([db-adapter attr-ident-or-rel-attr-ident eql-node group-by-column]
   (let [heql-meta-data    (:heql-meta-data db-adapter)
         attr-ident        (if (keyword? attr-ident-or-rel-attr-ident) attr-ident-or-rel-attr-ident (last attr-ident-or-rel-attr-ident))
         attr-md           (heql-md/attr-meta-data heql-meta-data attr-ident)
         attr-col-name     (:attr.column/name attr-md)
         attr-col-ref-type (:attr.column.ref/type attr-md)
         eql-node          (if (keyword? attr-ident-or-rel-attr-ident)
                             eql-node
                             (first (filter #(= (:key %) (first attr-ident-or-rel-attr-ident)) (:children eql-node))))
         {:keys [self parent]} (:alias eql-node)]
     (if (and group-by-column (= :attr.column.ref.type/one-to-one attr-col-ref-type))
       (resolve-group-by-column db-adapter eql-node attr-ident)
       (if (keyword? attr-ident-or-rel-attr-ident)
         (keyword (str self "." attr-col-name))
         (keyword (str parent "__" self)
                  (str (namespace (:attr/ident attr-md)) "/" (name (:attr/ident attr-md)))))))))

(defn- order-by-clause [db-adapter eql-node clause]
  (if (keyword? clause)
    (hsql-column db-adapter clause eql-node)
    (let [[c t]         clause]
      (if (#{:asc :desc} t)
        [(hsql-column db-adapter c eql-node) t]
        (hsql-column db-adapter clause eql-node)))))

(defn- apply-order-by [hsql heql-meta-data clause eql-node]
  (assoc hsql :order-by (map #(order-by-clause heql-meta-data eql-node %) clause)))

(defn- coerce-value [db-adapter eql-node col value]
  (if (coll? value)
    (map #(coerce-value db-adapter eql-node col %) value)
    (if (and (keyword? value) (some? (namespace value)) (heql-md/attribute? (:heql-meta-data db-adapter) value))
      (hsql-column db-adapter value eql-node)
      (heql-md/coerce-attr-value db-adapter :from-db col value))))

(defn- hsql-predicate [db-adapter eql-node clause]
  (let [[op col v1 v2] clause
        hsql-col       (hsql-column db-adapter col eql-node)]
    (if v2
      [op hsql-col (coerce-value db-adapter eql-node col v1) (coerce-value db-adapter eql-node col v2)]
      [op hsql-col (coerce-value db-adapter eql-node col v1)])))

(defn- hsql-join-predicate [db-adapter eql-node join-attr-md self-alias]
  (let [heql-meta-data      (:heql-meta-data db-adapter)
        ref-type            (:attr.column.ref/type join-attr-md)
        alias               {:self   self-alias
                             :parent (get-in eql-node [:alias :self])}
        from-relation-ident (heql-md/entity-relation-ident heql-meta-data (:attr.column.ref/right join-attr-md))
        from-clause         [from-relation-ident (keyword self-alias)]]
    (case ref-type
      :attr.column.ref.type/one-to-one [[from-clause]
                                        (one-to-*-join-predicate heql-meta-data join-attr-md alias)]
      :attr.column.ref.type/one-to-many [[from-clause]
                                         (one-to-*-join-predicate heql-meta-data join-attr-md alias)]
      :attr.column.ref.type/many-to-many (let [assoc-table-alias       (str (gensym))
                                               assoc-table-from-clause [(->> (:attr.column.ref.associative/ident join-attr-md)
                                                                             (heql-md/entity-meta-data heql-meta-data)
                                                                             :entity.relation/ident)
                                                                        (keyword assoc-table-alias)]]
                                           [[from-clause assoc-table-from-clause]
                                            (many-to-many-join-predicate heql-meta-data join-attr-md alias assoc-table-alias)]))))

(defn- nested-entity-attr-predicate [db-adapter eql-node clause hsql join-attr-md attr-ident]
  (let [[op _ v1 v2] clause
        attr-ident   (if (qualified-keyword? attr-ident)
                       attr-ident
                       (keyword (name (:attr.ref/type join-attr-md)) (name attr-ident)))]
    (hsql-helpers/where
     hsql
     (hsql-predicate db-adapter eql-node [op attr-ident v1 v2]))))

(defn- nested-entity-predicate [db-adapter eql-node clause]
  (let [[op col]         clause
        join-attr-ident  (if (= :exists op) col (first col))
        attr-ident       (when-not (= :exists op) (second col))
        heql-meta-data   (:heql-meta-data db-adapter)
        self-alias       (str (gensym))
        self-eql-node    {:alias {:self self-alias}}
        join-attr-md     (heql-md/attr-meta-data heql-meta-data join-attr-ident)
        [from join-pred] (hsql-join-predicate db-adapter eql-node join-attr-md self-alias)
        hsql             {:select [1]
                          :from   from
                          :where  join-pred}]
    [:exists (if attr-ident
               (nested-entity-attr-predicate db-adapter self-eql-node clause hsql join-attr-md attr-ident)
               hsql)]))

(defn- where-predicate [db-adapter clause eql-node]
  (let [[op col] clause]
    (case op
      :and (concat [:and] (map #(where-predicate db-adapter % eql-node) (rest clause)))
      :or (concat [:or] (map #(where-predicate db-adapter % eql-node) (rest clause)))
      :not (conj [:not] [(where-predicate db-adapter (second clause) eql-node)])
      :exists (nested-entity-predicate db-adapter eql-node clause)
      (if (keyword? col)
        (hsql-predicate db-adapter eql-node clause)
        (nested-entity-predicate db-adapter eql-node clause)))))

(defn- apply-where [hsql db-adapter clause eql-node]
  (hsql-helpers/where hsql (where-predicate db-adapter clause eql-node)))

(defn- apply-group-by [hsql db-adapter clause eql-node]
  (apply hsql-helpers/group-by hsql (map #(hsql-column db-adapter % eql-node true) clause)))

(defn- apply-params [db-adapter hsql eql-node]
  (let [{:keys [limit offset order-by where group-by]} (:params eql-node)]
    (cond-> hsql
      limit  (assoc :limit limit)
      offset (assoc :offset offset)
      order-by (apply-order-by db-adapter order-by eql-node)
      where (apply-where db-adapter where eql-node)
      group-by (apply-group-by db-adapter group-by eql-node)
      :else identity)))

(defmethod ^{:private true} dsl/eql->hsql :ident-join [db-adapter heql-meta-data eql-node]
  (let [{:keys [key children alias]} eql-node
        hsql                         {:from   [[(heql-md/entity-relation-ident heql-meta-data (first key))
                                                (keyword (:self alias))]]
                                      :where  (eql-ident-key->hsql-predicate db-adapter key alias)
                                      :select (db/select-clause db-adapter heql-meta-data children)}
        hsql                         (apply-params db-adapter hsql eql-node)]
    (db/resolve-children-one-to-one-relationships db-adapter heql-meta-data hsql children)))

(defmethod ^{:private true} dsl/eql->hsql :non-ident-join [db-adapter heql-meta-data eql-node]
  (let [{:keys [children alias]} eql-node
        first-child-ident        (eql-node->attr-ident (first children))
        hsql                     {:from   [[(heql-md/entity-relation-ident heql-meta-data first-child-ident)
                                            (keyword (:self alias))]]
                                  :select (db/select-clause db-adapter heql-meta-data children)}
        hsql                     (apply-params db-adapter hsql eql-node)]
    (db/resolve-children-one-to-one-relationships db-adapter heql-meta-data hsql children)))

(defmethod ^{:private true} dsl/eql->hsql :one-to-one-join [db-adapter heql-meta-data eql-node]
  (let [{:keys [key children alias]} eql-node
        key                          (if (dsl/alias-attribute-ident? key)
                                       (first key)
                                       key)
        join-attr-md                 (heql-md/attr-meta-data heql-meta-data key)
        hsql                         {:from   [[(heql-md/ref-entity-relation-ident heql-meta-data key)
                                                (keyword (:self alias))]]
                                      :where  (one-to-*-join-predicate heql-meta-data join-attr-md alias)
                                      :select (db/select-clause db-adapter heql-meta-data children)}
        hsql                         (apply-params db-adapter hsql eql-node)]
    (db/resolve-one-to-one-relationship db-adapter heql-meta-data hsql eql-node)))

(defmethod ^{:private true} dsl/eql->hsql :one-to-many-join [db-adapter heql-meta-data eql-node]
  (let [{:keys [key children alias]} eql-node
        key                          (if (dsl/alias-attribute-ident? key)
                                       (first key)
                                       key)
        join-attr-md                 (heql-md/attr-meta-data heql-meta-data key)
        hsql                         {:from   [[(heql-md/ref-entity-relation-ident heql-meta-data key)
                                                (keyword (:self alias))]]
                                      :where  (one-to-*-join-predicate heql-meta-data join-attr-md alias)
                                      :select (db/select-clause db-adapter heql-meta-data children)}
        hsql                         (apply-params db-adapter hsql eql-node)]
    (db/resolve-one-to-many-relationship db-adapter heql-meta-data hsql eql-node)))

(defmethod ^{:private true} dsl/eql->hsql :many-to-many-join [db-adapter heql-meta-data eql-node]
  (let [{:keys [key children alias]} eql-node
        key                          (if (dsl/alias-attribute-ident? key)
                                       (first key)
                                       key)
        join-attr-md                 (heql-md/attr-meta-data heql-meta-data key)
        assoc-table-alias            (gensym)
        hsql                         {:from   [[(heql-md/ref-entity-relation-ident heql-meta-data key)
                                                (keyword (:self alias))]
                                               [(->> (:attr.column.ref.associative/ident join-attr-md)
                                                     (heql-md/entity-meta-data heql-meta-data)
                                                     :entity.relation/ident)
                                                (keyword assoc-table-alias)]]
                                      :where  (many-to-many-join-predicate heql-meta-data join-attr-md alias assoc-table-alias)
                                      :select (db/select-clause db-adapter heql-meta-data children)}
        hsql                         (apply-params db-adapter hsql eql-node)]
    (db/resolve-many-to-many-relationship db-adapter heql-meta-data hsql eql-node)))
