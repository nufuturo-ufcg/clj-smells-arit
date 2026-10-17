(ns com.walmartlabs.lacinia.schema
  "Responsible for constructing and validating the GraphQL schema.

  GraphQL schema starts in a format easy to read and maintain as an EDN file.

  Compiling the schema performs a number of validations and reorganizations to
  make query execution faster and simpler, such as generating a flatter structure for the
  schema, and pre-computing many defaults."
  (:refer-clojure :exclude [compile])
  (:require
    [clojure.spec.alpha :as s]
    [com.walmartlabs.lacinia.introspection :as introspection]
    [com.walmartlabs.lacinia.internal-utils
     :refer [map-vals map-kvs filter-vals deep-map-merge q get-nested
             is-internal-type-name? sequential-or-set? as-keyword
             ->TaggedValue is-tagged-value? extract-value extract-type-tag
             to-message qualified-name fast-map-indexed]]
    [better-cond.core :as b]
    [com.walmartlabs.lacinia.select-utils :as su]
    [com.walmartlabs.lacinia.resolve :as resolve
     :refer [ResolverResult resolve-as is-resolver-result?]]
    [com.walmartlabs.lacinia.resolve-utils :refer [aggregate-results]]
    [clojure.string :as str]
    [clojure.set :refer [difference]]
    [clojure.pprint :as pprint]
    [com.walmartlabs.lacinia.selection :as selection])
  (:import
    (clojure.lang IObj PersistentQueue)
    (java.io Writer)
    (java.util.concurrent Executor ThreadPoolExecutor TimeUnit LinkedBlockingQueue ThreadFactory)))

(defmulti ^:private check-compatible
  "Given two type definitions, dispatches on a vector of the category of the two types.
  'Returns true if the two types are compatible.

  The interface defines the constraint type, the field defines the constrained type.

  This is only invoked when the constraint type and constrained types are not equal.

  The rules for this are in section 3.1.2.3 of the spec."
  (fn [constraint-type constrained-type]
    (mapv :category [constraint-type constrained-type])))

(defmethod check-compatible :default
  [_ _]
  ;; Remember that for object-vs-object, scalar-vs-scalar, and
  ;; enum-vs-enum, we don't get this far if the types are the same.
  ;; For disparate types, generally not compatible (e.g., enum vs. scalar).
  false)

(defmethod check-compatible [:union :object]
  [i-type f-type]
  (contains? (:members i-type) (:type-name f-type)))

(defmethod check-compatible [:interface :object]
  [i-type f-type]
  (contains? (:implements f-type) (:type-name i-type)))

(defmethod check-compatible [:interface :interface]
  [i-type f-type]
  ;; An interface field type is compatible with an implementing-interface field type if
  ;; the implementing interface (f-type) declares that it implements the constraining
  ;; interface (i-type).
  (contains? (:implements f-type) (:type-name i-type)))

(def ^{:dynamic true
       :added   "0.25.0"} *verbose-schema-printing*
  "If bound to true, then the compiled schema prints and pretty-prints like an ordinary map,
  which is sometimes useful during development. When false (the default) the schema
  output is just a placeholder."
  false)
