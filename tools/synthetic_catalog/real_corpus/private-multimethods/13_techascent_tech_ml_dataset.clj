(ns tech.v3.libs.arrow
  "Support for reading/writing apache arrow datasets.  Datasets may be memory mapped
  but default to being read via an input stream.

  Supported datatypes:

  * All numeric types - `:uint8`, `:int8`, `:uint16`, `:int16`, `:uint32`, `:int32`,
  `:uint64`, `:int64`, `:float32`, `:float64`, `:boolean`.
  * String types - `:string`, `:text`.  During write you have the option to always write
  data as text which can be more efficient in the memory-mapped read case as it doesnt'
  require the creation of string tables at load time.
  * Datetime Types - `:local-date`, `:local-time`, `:instant`.  During read you have the
  option to keep these types in their source numeric format e.g. 32 bit `:epoch-days`
  for `:local-date` datatypes.  This format can make some types of processing, such as
  set creation, more efficient.


  When writing a dataset an arrow file with a single record set is created.  When
  writing a sequence of datasets downstream schemas must be compatible with the schema
  of the initial dataset so for instance a conversion of int32 to double is fine but
  double to int32 is not.

  mmap support on systems running JDK-17 requires the foreign or memory module to be
  loaded.  Appropriate JVM arguments can be found
  [here](https://github.com/techascent/tech.ml.dataset/blob/0524ddd5bbcb9421a0f11290ec8a01b7795dcff9/project.clj#L69).

  Example (with zstd compression):

```clojure
  ;; Writing
  (arrow/dataset->stream! ds fname {:compression :zstd})
  ;; Reading
  (arrow/stream->dataset path)
```

  ## Required Dependencies

  In order to support both memory mapping and JDK-17, we only rely on the Arrow SDK's
  flatbuffer and schema definitions:

```clojure
  ;; netty isn't required and will inevitably conflict with some more recent version
  [org.apache.arrow/arrow-vector \"6.0.0\":exclusions [netty/netty io.netty/netty-common]]
  [com.cnuernber/jarrow \"1.000\"]
  [org.apache.commons/commons-compress \"1.21\"]

  ;;Compression codecs
  [org.lz4/lz4-java \"1.8.0\"]
  ;;Required for decompressing lz4 streams with dependent blocks.
  [net.java.dev.jna/jna \"5.10.0\"]
  [com.github.luben/zstd-jni \"1.5.4-1\"]
```
  The lz4 decompression system will fallback to lz4-java if liblz4 isn't installed or if
  jna isn't loaded.  The lz4-java java library will fail for arrow files that have dependent
  block compression which are sometimes saved by python or R arrow implementations.
  On current ubuntu, in order to install the lz4 library you need to do:

```console
  sudo apt install liblz4-1
  ```  

  ## Performance 

  Arrow has hands down highest performance of any of the formats although nippy comes very close when using
  any compression.  The highest performance pathway is to save out data with :strings-as-text? true and zero
  compression then read them in using mmap - optionally with :text-as-strings? if you never want to see 
  tech.v3.datatype.Text objects in your dataset.  This avoids the creation of string dictionaries during 
  deserialization as these have to be done greedily.  It can dramatically increase many dataset sizes but
  when mmap is used the overall size is irrelevant aside from iteration which can be heavily parallelized.

  Example:

```clojure
  ;; Writing
  (arrow/dataset->stream! ds fname {:strings-as-text? true})
  ;; Reading
  (arrow/stream->dataset path {:text-as-strings? true :open-type :mmap})
  ```"
  (:require [tech.v3.datatype.mmap :as mmap]
            [tech.v3.datatype.datetime :as dtype-dt]
            [tech.v3.datatype :as dtype]
            [tech.v3.datatype.protocols :as dtype-proto]
            [tech.v3.datatype.native-buffer :as native-buffer]
            [tech.v3.datatype.nio-buffer :as nio-buffer]
            [tech.v3.datatype.bitmap :as bitmap]
            [tech.v3.datatype.casting :as casting]
            [tech.v3.datatype.packing :as packing]
            [tech.v3.datatype.array-buffer :as array-buffer]
            [tech.v3.datatype.ffi :as dt-ffi]
            [tech.v3.dataset.impl.column :as col-impl]
            [tech.v3.dataset.impl.sparse-column :as sparse-col]
            [tech.v3.dataset.impl.column-base :as col-base]
            [tech.v3.dataset.column :as ds-col]
            [tech.v3.dataset.protocols :as ds-proto]
            [tech.v3.dataset.impl.dataset :as ds-impl]
            [tech.v3.dataset.dynamic-int-list :as dyn-int-list]
            [tech.v3.dataset.base :as ds-base]
            [tech.v3.dataset.string-table :as str-table]
            [tech.v3.dataset.utils :as ml-utils]
            [tech.v3.dataset.io :as ds-io]
            [tech.v3.parallel.for :as pfor]
            [tech.v3.resource :as resource]
            [tech.v3.io :as io]
            [clojure.tools.logging :as log]
            [clj-commons.primitive-math :as pmath]
            [clojure.core.protocols :as clj-proto]
            [clojure.datafy :refer [datafy]]
            [charred.api :as json]
            [ham-fisted.api :as hamf]
            [ham-fisted.reduce :as hamf-rf]
            [ham-fisted.lazy-noncaching :as lznc]
            [ham-fisted.function :as hamf-fn]
            [ham-fisted.set :as set]
            [ham-fisted.protocols :as hamf-proto])
  (:import [ham_fisted ArrayLists ArrayLists$ArrayOwner IMutList]
           [org.apache.arrow.vector.ipc.message MessageSerializer]
           [org.apache.arrow.flatbuf Message DictionaryBatch RecordBatch
            FieldNode Buffer BodyCompression BodyCompressionMethod Footer Block]
           [org.roaringbitmap RoaringBitmap]
           [com.google.flatbuffers FlatBufferBuilder]
           [org.apache.arrow.vector.types TimeUnit FloatingPointPrecision DateUnit]
           [org.apache.arrow.vector.types.pojo Field Schema ArrowType$Int
            ArrowType$Utf8 ArrowType$Timestamp ArrowType$Time DictionaryEncoding FieldType
            ArrowType$FloatingPoint ArrowType$Bool ArrowType$Date ArrowType$Duration
            ArrowType$LargeUtf8 ArrowType$Null ArrowType$List ArrowType$Binary ArrowType$FixedSizeBinary
            ArrowType$Decimal]
           [org.apache.arrow.flatbuf CompressionType]
           [org.apache.arrow.vector.types MetadataVersion]
           [org.apache.arrow.vector.ipc WriteChannel]           
           [tech.v3.dataset.string_table StringTable]
           [tech.v3.dataset.impl.column Column]
           [tech.v3.dataset Text]
           [tech.v3.datatype.native_buffer NativeBuffer]
           [tech.v3.datatype ObjectReader ArrayHelpers ByteConversions BooleanBuffer]
           [tech.v3.datatype.array_buffer ArrayBuffer]
           [java.io OutputStream InputStream ByteArrayOutputStream ByteArrayInputStream]
           [java.nio ByteBuffer ByteOrder ShortBuffer IntBuffer LongBuffer DoubleBuffer
            FloatBuffer]
           [java.util List ArrayList Map HashMap Map$Entry Iterator Set UUID Arrays]
           [java.util.concurrent ForkJoinTask]
           [java.time ZoneId]
           [java.nio.channels WritableByteChannel]
           ;;Compression codecs
           [com.github.luben.zstd Zstd]
           [org.apache.commons.compress.compressors.lz4 FramedLZ4CompressorInputStream
            FramedLZ4CompressorOutputStream]
           ;;feather support
           [java.io RandomAccessFile BufferedInputStream ByteArrayInputStream
            ByteArrayOutputStream]
           [uk.ac.bristol.star.feather FeatherTable BufUtils
            FeatherType]
           [uk.ac.bristol.star.fbs.feather Type]
           [uk.ac.bristol.star.fbs.feather CTable]))

(defn- field-type
  ^FieldType
  ([nullable? ^FieldType datatype ^DictionaryEncoding dict-encoding ^Map str-str-meta]
   (FieldType. (boolean nullable?) datatype dict-encoding str-str-meta))
  ([nullable? datatype str-meta]
   (field-type nullable? datatype nil str-meta))
  ([nullable? datatype]
   (field-type nullable? datatype nil)))

(defn- try-json-parse
  [val]
  (when val
    (try (json/read-json val :key-fn keyword)
         (catch Exception e
           val))))

(defn- datafy-field
  [^Field field]
  (let [df (datafy (.. field getFieldType getType))]
    (when-not (map? df)
      (throw (Exception. (format "Failed to dataty datatype %s-%s"
                                 field (type (.. field getFieldType getType))))))
    (merge
     {:name (.getName field)
      :nullable? (.isNullable field)
      :field-type df
      :metadata (->> (.getMetadata field)
                     (map (fn [^Map$Entry entry]
                            [(try-json-parse (.getKey entry))
                             (try-json-parse (.getValue entry))]))
                     (into {}))}
     (when-let [c (seq (.getChildren field))]
       {:children (mapv datafy-field c)})
     (when-let [encoding (.getDictionary field)]
       {:dictionary-encoding (datafy encoding)}))))

(defn- read-schema
  "returns a pair of offset-data and schema"
  [{:keys [message _body _message-type]}]
  (let [schema (MessageSerializer/deserializeSchema ^Message message)
        ;; _ (println schema)
        fields
        (->> (.getFields schema)
             (mapv datafy-field))
        metadata (when-let [metadata (.getCustomMetadata schema)]
                   (into {} metadata))
        sparse-columns (when-let [sparse-columns (get metadata "sparse-columns")]
                         (->> (.split ^String sparse-columns ",")
                              (map #(Long/parseLong %))
                              (hamf/java-hashset)))]
    {:fields fields
     :encodings (->> (map :dictionary-encoding fields)
                     (remove nil?)
                     (map (juxt :id identity))
                     (into {}))
     :metadata metadata
     :sparse-columns sparse-columns}))

(defn- read-record-batch
  ([^RecordBatch record-batch ^NativeBuffer data]
   {:compression (.compression record-batch)
    :nodes (->> (hamf/range (.nodesLength record-batch))
                (mapv #(let [node (.nodes record-batch (unchecked-int %))]
                         {:n-elems (.length node)
                          :n-null-entries (.nullCount node)})))
    :buffers (->> (hamf/range (.buffersLength record-batch))
                  (mapv #(let [buffer (.buffers record-batch (unchecked-int %))]
                           (dtype/sub-buffer data (.offset buffer)
                                             (.length buffer)))))})
  ([{:keys [message body _message-type]}]
   (read-record-batch (.header ^Message message (RecordBatch.)) body)))

(defn- check-message-type
  [expected-type actual-type]
  (when-not (= actual-type expected-type)
    (throw (Exception.
            (format "Expected message type %s, got %s"
                    expected-type actual-type)))))

(defn- read-dictionary-batch
  [{:keys [message body message-type]}]
  (check-message-type :dictionary-batch message-type)
  (let [^DictionaryBatch db (.header ^Message message (DictionaryBatch.))]
    {:id (.id db)
     :delta? (.isDelta db)
     :records (read-record-batch (.data db) body)}))

(defmulti ^:private parse-message
  "Given a message, parse it just a bit into a more interpretable datastructure."
  :message-type)

(defmethod ^:private parse-message :schema
  [msg]
  (assoc (read-schema msg)
         :message-type (:message-type msg)))

(defmethod ^:private parse-message :dictionary-batch
  [msg]
  (assoc (read-dictionary-batch msg)
         :message-type (:message-type msg)))

(defmethod ^:private parse-message :record-batch
  [msg]
  (assoc (read-record-batch msg)
         :message-type (:message-type msg)))
