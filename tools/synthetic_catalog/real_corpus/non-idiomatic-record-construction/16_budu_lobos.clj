(ns lobos.schema
  "This namespace include the abstract schema data-structures, an
  handful of helpers to create them and the protocol to build the into
  an abstract syntax tree of implementation agnostic SQL
  statements. Abstract schema data-structures can be divided in two
  categories.

  First, schema elements which include the `schema` and the `table`
  element definitions. Those can be created or dropped, also the `table`
  element can be altered.

  Then there's the table elements that serves to define tables. There's
  the completly abstract `column` and `constraint` elements, which are
  only meant to be used directly with the alter action. Each of them
  have more specialized function to help you define tables, like the
  `unique`, `primary-key`, `foreign-key` and `check` constraints
  definitons and the typed data definitions."
  (:refer-clojure :exclude [defonce replace
                            bigint boolean char double float time])
  (:require (lobos [ast :as ast]))
  (:use (clojure [walk   :only [postwalk]]
                 [set    :only [union]]
                 [string :only [replace]])
        lobos.utils))

(defprotocol Alterable
  "The Alterable protocol add the possibility of building alter
  statements from an object implementing it. *For internal use*."
  (build-alter-statement [this action db-spec]))

(defprotocol Buildable
  "The Buildable protocol is currently used only by table elements.
  *For internal use*."
  (build-definition [this db-spec]))

(defprotocol Creatable
  "The Creatable protocol add the possibility of building create
  statements from an object implementing it. *For internal use*."
  (build-create-statement [this db-spec]))

(defprotocol Dropable
  "The Dropable protocol add the possibility of building drop
  statements from an object implementing it. *For internal use*."
  (build-drop-statement [this behavior db-spec]))

(defn name-required
  "Throws an IllegalArgumentException when the given name is nil with a
  default message using the given type of elements."
  [name etype]
  (when-not name
    (throw (IllegalArgumentException.
            (format "A %s definition needs at least a name."
                    etype)))))

(def sql-symbols
  (union sql-infix-operators
         sql-prefix-operators
         sql-functions))

(defrecord Expression [value]
  Buildable

  (build-definition [this db-spec]
    (postwalk
     #(do
        (cond (vector? %)
              (let [[f & n] %]
                (if (keyword? f)
                  (condp contains? (-> f name symbol)
                    sql-infix-operators
                    (OperatorExpression. db-spec f (first n) (next n))
                    sql-prefix-operators
                    (OperatorExpression. db-spec f nil n)
                    sql-functions
                    (FunctionExpression. db-spec f n))
                  %))
              (and (keyword? %)
                   (not (contains? sql-symbols (-> % name symbol))))
              (IdentifierExpression. db-spec % nil)
              (not (keyword? %))
              (ScalarExpression. db-spec %)
              :else %))
     value)))

(defmacro expression [form]
  `(Expression.
    ~(postwalk
      #(if (and (seq? %)
                (sql-symbols (first %)))
         (apply vector
                (keyword (first %))
                (rest %))
         %)
      form)))

(defrecord DataType [name args options])

(defn data-type
  "Constructs an abstract data-type definition using the given keyword
  `name`. Can also take an options list of arguments (`args`) and
  `options`."
  [name & [args options]]
  (check-valid-options options :encoding :collate :time-zone)
  (update-options
   (DataType. name (vec args) {})
   options))

(defn datetime-now-alias
  "If the given default value, it will be replaced by the standard
  function returning the current time, date or timestamp depending on
  the specified data-type. *For internal use*."
  [name default]
  (let [value (:value default)]
    (if (= value [:now])
      (Expression.
       (or ({:date [:current_date]
             :time [:current_time]
             :timestamp [:current_timestamp]} name) value))
      default)))

(defmacro default [form]
  `[:default (expression ~form)])

(defn- build-table-elements [db-spec method & elements]
  (->> (apply concat elements)
       (map #(when (second %)
               (method (second %) db-spec)))
       (filter identity)))

(defrecord Table [name columns constraints indexes]
  Alterable Creatable Dropable

  (build-alter-statement [this action db-spec]
    (let [elements (build-table-elements db-spec
                                         build-definition
                                         columns
                                         constraints)]
      (for [element elements]
        (AlterTableStatement.
         db-spec
         name
         action
         element))))

  (build-create-statement [this db-spec]
    (conj
     (build-table-elements db-spec build-create-statement indexes)
     (CreateTableStatement.
      db-spec
      name
      (build-table-elements db-spec build-definition columns constraints))))

  (build-drop-statement [this behavior db-spec]
    (DropStatement. db-spec :table name behavior nil)))

(defn table*
  "Constructs an abstract table definition. The `table-name` is
  mandatory."
  [name & [columns constraints indexes]]
  (name-required name "table")
  (Table. name
          (or columns {})
          (or constraints {})
          (or indexes {})))

(defmacro table
  "Constructs an abstract table definition containing the given
  elements."
  [name & elements]
  `(-> (table* ~name) ~@(reverse elements)))
