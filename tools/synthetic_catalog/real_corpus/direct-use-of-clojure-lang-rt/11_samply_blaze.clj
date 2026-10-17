(ns blaze.fhir.spec.resource
  "Streaming reader and writer for FHIR resources and complex types.

  Reads and writes Blaze's internal FHIR data model from/to both JSON and CBOR,
  on top of Jackson's streaming `JsonParser` / `JsonGenerator` (the CBOR backend
  implements the same streaming interfaces). Parsing produces values directly
  from the token stream without an intermediate tree, so a resource is built in
  a single pass.

  Use `create-type-handlers` to build a map of type-handlers from a
  StructureDefinition snapshot, then pass it to `parse-json` / `parse-cbor`
  (read) or `write-json` / `write-cbor` (write).

  A locator is a list of path segments already parsed in order to report the
  location of an error. Path segments are either strings of field names or
  indices of arrays. The lists are built in reverse where the path grows at the
  front. In case of an error, the locator list is reversed.

  A type-handler in this namespace is a function of two arities. On arity-0 the
  function returns the name of the type as string. On arity-3 it takes a map of
  all type handlers, a parser and a locator and returns either a FHIR value or
  an anomaly in case of errors.

  A property-handler in this namespace is a function taking a map of all type
  handlers, a parser, a locator and a partially constructed FHIR value and
  returns either the FHIR value with data added or an anomaly in case of errors.

  Summary variant: when `create-type-handlers` is called with
  `:include-summary-only true`, it builds a second, summary-only handler for
  every type (keyed under the `summary` namespace, e.g. `:summary/Patient`). A
  summary handler only has property-handlers for elements marked `isSummary` in
  the StructureDefinition; every other property has no handler and is skipped in
  place by `skip-value!` (Jackson's `.skipChildren`), so non-summary subtrees are
  walked past in the token stream but never materialized into FHIR values.
  Because every backbone element gets its own summary handler too, a non-summary
  child of a summary element (e.g. `Observation.component.referenceRange`) is
  skipped as well. `parse-cbor` selects this variant via its `variant` argument;
  the resulting resource is tagged with the SUBSETTED meta tag.

  This namespace uses some advanced optimizations like mutable ArrayLists.
  Please change with care."
  (:refer-clojure :exclude [str])
  (:require
   [blaze.anomaly :as ba :refer [if-ok when-ok]]
   [blaze.fhir.spec.type :as type]
   [blaze.fhir.spec.type.string-util :as su]
   [blaze.fhir.spec.type.system :as system]
   [blaze.fhir.util :as fu]
   [blaze.util :as u :refer [str]]
   [clojure.string :as str]
   [cognitect.anomalies :as anom])
  (:import
   [blaze.fhir.spec.type Lists]
   [clojure.lang PersistentArrayMap RT]
   [com.fasterxml.jackson.core JsonFactory JsonParseException JsonParser JsonToken StreamReadConstraints]
   [com.fasterxml.jackson.core.exc InputCoercionException]
   [com.fasterxml.jackson.core.io JsonEOFException]
   [com.fasterxml.jackson.databind JsonNode ObjectMapper]
   [com.fasterxml.jackson.databind.node TreeTraversingParser]
   [com.fasterxml.jackson.dataformat.cbor CBORFactory]
   [java.io InputStream OutputStream Reader]
   [java.util ArrayList Arrays List]))

(defmacro current-token [parser]
  `(.currentToken ~(with-meta parser {:tag `JsonParser})))

(defn- expression [locator]
  ;; the locator is reversed because it's a list were path segments are appended
  ;; in front
  (let [locator (reverse locator)]
    (loop [sb (StringBuilder. (str (first locator)))
           more (next locator)]
      (if more
        (if (string? (first more))
          (recur (-> sb (.append ".") (.append (first more))) (next more))
          (recur (-> sb (.append "[") (.append (str (first more))) (.append "]"))
                 (next more)))
        (str sb)))))

(defn- fhir-issue [msg locator]
  {:fhir.issues/code "invariant"
   :fhir.issues/diagnostics msg
   :fhir.issues/expression (expression locator)})

(defn- unexpected-end-of-input-msg [^JsonEOFException e]
  (condp identical? (.getTokenBeingDecoded e)
    JsonToken/FIELD_NAME "Unexpected end of input while parsing a field name."
    "Unexpected end of input."))

(defn- next-token! [parser locator]
  (try
    (.nextToken ^JsonParser parser)
    (catch JsonEOFException e
      (let [msg (unexpected-end-of-input-msg e)]
        (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)])))
    (catch JsonParseException _
      (let [msg "JSON parsing error."]
        (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)])))))

(defn- get-text [parser locator]
  (try
    (.getText ^JsonParser parser)
    (catch JsonEOFException _
      (let [msg "Unexpected end of input while reading a string value."]
        (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)])))))

(defn- get-long [parser locator]
  (try
    (.getLongValue ^JsonParser parser)
    (catch InputCoercionException e
      (let [msg (first (str/split-lines (ex-message e)))]
        (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)])))))

(defn- get-current-value [parser locator]
  (condp identical? (current-token parser)
    JsonToken/VALUE_NULL "value null"
    JsonToken/VALUE_TRUE "boolean value true"
    JsonToken/VALUE_FALSE "boolean value false"
    JsonToken/VALUE_STRING
    (when-ok [text (get-text parser locator)]
      (format "value `%s`" text))
    JsonToken/VALUE_NUMBER_INT
    (when-ok [number (get-long parser locator)]
      (format "integer value %d" number))
    JsonToken/VALUE_NUMBER_FLOAT
    (format "float value %s" (.getDecimalValue ^JsonParser parser))
    JsonToken/START_OBJECT "object start"
    JsonToken/START_ARRAY "array start"
    (format "token %s" (current-token parser))))

(defn- incorrect-value-anom*
  ([value locator expected-type]
   (let [msg (format "Error on %s. Expected type is `%s`." value expected-type)]
     (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)])))
  ([value locator expected-type reason-msg]
   (let [msg (format "Error on %s. Expected type is `%s`. %s" value expected-type reason-msg)]
     (ba/incorrect msg :fhir/issues [(fhir-issue msg locator)]))))

(defn- incorrect-value-anom [parser locator expected-type]
  (when-ok [value (get-current-value parser locator)]
    (incorrect-value-anom* value locator expected-type)))

(defmacro cond-next-token [parser locator & body]
  `(when-ok [token# (next-token! ~parser ~locator)]
     (condp identical? token#
       ~@body)))

(defn- unsupported-type-anom [type]
  (ba/unsupported (format "Unsupported type `%s`." type)))

(defn- read-value* [type-handlers parser locator handler]
  (cond-next-token parser locator
    JsonToken/START_OBJECT
    (when-ok [type (handler type-handlers parser locator)
              token (next-token! parser locator)]
      (if token
        (ba/incorrect (format "incorrect trailing token %s" token))
        type))
    (incorrect-value-anom parser locator (handler))))

(defn- prefix-msg [msg]
  (str "Invalid JSON representation of a resource. " msg))

(defn- read-value
  "Reads a complex value from `parser` using `handler`.

  The locator is used to generate path-based expressions in errors. The initial
  locator has to be a list not a vector.

  The handler will determine the type of the value."
  [type-handlers parser locator handler]
  (-> (read-value* type-handlers parser locator handler)
      (ba/exceptionally #(update % ::anom/message prefix-msg))))

(def ^:private stream-read-constraints
  (-> (StreamReadConstraints/builder)
      (.maxStringLength Integer/MAX_VALUE)
      (.build)))

(def ^:private ^JsonFactory cbor-factory
  (doto (-> (CBORFactory/builder)
            (.streamReadConstraints stream-read-constraints)
            (.build))
    (ObjectMapper.)))

(defn parse-cbor
  "Parses a complex value of `type` and `variant` from CBOR `source`.

  `variant` is `:summary` to parse only the summary elements (using the
  `summary`-namespaced handler built with `:include-summary-only`; non-summary
  elements are skipped in the token stream without being materialized) or any
  other value (e.g. `:complete`) to parse the full value.

  Returns an anomaly in case of errors."
  [type-handlers type variant source]
  (if-some [handler (get type-handlers (if (= :summary variant) (keyword "summary" type) (keyword type)))]
    (with-open [parser (.createParser ^JsonFactory cbor-factory ^bytes source)]
      (read-value type-handlers parser (RT/list type) handler))
    (unsupported-type-anom type)))
