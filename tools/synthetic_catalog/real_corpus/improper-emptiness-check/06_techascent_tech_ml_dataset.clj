(ns tech.v3.libs.parquet-hardwood
  (:require [ham-fisted.lazy-caching :as lznc]
            [ham-fisted.api :as hamf]
            [ham-fisted.iterator :as hamf-iter]
            [tech.v3.datatype :as dt]
            [tech.v3.datatype.array-buffer :as array-buffer]
            [tech.v3.datatype.errors :as errors]
            [tech.v3.dataset :as ds]
            [tech.v3.dataset.io.column-parsers :as col-parsers]
            [tech.v3.dataset.io.context :as parse-context]
            [tech.v3.dataset.base :as ds-base]
            [clojure.java.io :as io]
            [clojure.tools.logging :as log])
  (:import [dev.morling.hardwood.reader ParquetFileReader ColumnReader]
           [dev.morling.hardwood.schema FileSchema ColumnSchema]
           [dev.morling.hardwood.metadata RowGroup ColumnChunk LogicalType
            LogicalType$DateType LogicalType$StringType LogicalType$UuidType
            LogicalType$EnumType LogicalType$JsonType LogicalType$TimestampType
            LogicalType$BsonType LogicalType$IntervalType LogicalType$IntType
            LogicalType$DecimalType LogicalType$TimeType LogicalType$MapType
            LogicalType$ListType]
           [clojure.lang MapEntry]))

(def ^:private column-finalizers
  {LogicalType$StringType (fn [_ str-data] (dt/emap (fn [ss] (String. ^bytes ss)) :string str-data))
   LogicalType$DateType (fn [_ int-data] (-> (dt/as-array-buffer int-data)
                                             (array-buffer/set-datatype :packed-local-date)))})

(defn- read-column
  [idx ^ColumnSchema col-schema ^ColumnChunk col-def ^ParquetFileReader rdr options]
  (let [crdr (.getColumnReader rdr col-schema col-def)
        parser (col-parsers/promotional-object-parser (.name col-schema) nil)
        lt (.logicalType col-schema)
        finalizer (get column-finalizers (type lt) (fn [_ v] v))
        key-fn (:key-fn options identity)]
    (loop [idx 0]
      (if (.hasNext crdr)
        (do
          (when-let [nv (.readNext crdr)] (col-parsers/add-value! parser idx nv))
          (recur (inc idx)))
        {:column-name (key-fn (.name col-schema))
         :column-parser (reify tech.v3.dataset.io.column_parsers.PParser
                          (finalize [this n-rows]
                            (-> (col-parsers/finalize! parser n-rows)
                                (update :tech.v3.dataset/data #(finalizer lt %)))))
         :n-rows idx}))))

(defn parquet->ds-seq
  [path options]
  (let [rdr (ParquetFileReader/open (->path path))
        schema (.getFileSchema rdr)
        schema-columns (->> (range (.getColumnCount schema))
                            (into [] (map #(.getColumn schema (long %)))))
        col-name->idx (into {} (map-indexed (fn [idx ^ColumnSchema col]
                                              [(.name col) idx]))
                            schema-columns)
        col-idx->name (into {} (map-indexed (fn [idx ^ColumnSchema col]
                                              [idx (.name col)]))
                            schema-columns)
        cref->idx (fn [centry] (if (number? centry) (long centry) (col-name->idx centry)))
        allowset (into #{} (map cref->idx) (get options :column-allowlist))
        blockset (into #{} (map cref->idx) (get options :column-blocklist))
        is-allowed? (fn [^long idx]
                      (if-not (empty? allowset)
                        (boolean (allowset idx))
                        (if-not (empty? blockset)
                          (boolean (not (blockset idx)))
                          true)))
        src-indexes (vec (->> (hamf/range (count schema-columns))
                              (lznc/filter is-allowed?)))
        rv (->> (.getFileMetaData rdr)
             (.rowGroups)
             (lznc/map (fn [^RowGroup grp]
                         (let [parsers (->> src-indexes
                                            (hamf/pmap-io 12 (fn [^long idx]
                                                               (read-column idx (nth schema-columns idx)
                                                                            (nth (.columns grp) idx)
                                                                            rdr
                                                                            options)))
                                            (vec))
                               row-count (reduce max 0 (lznc/map :n-rows parsers))]
                           (parse-context/parsers->dataset options parsers row-count)))))
        closer* (delay (.close rdr))
        seq* (delay (seq rv))]
    (reify
      java.lang.AutoCloseable
      (close [this] @closer*)
      clojure.lang.Seqable
      (seq [this] @seq*)
      Iterable
      (iterator [this]
        (if (realized? seq*)
          (.iterator ^Iterable @seq*)
          (let [src-iter (.iterator ^Iterable rv)]
            (reify
              java.util.Iterator
              (hasNext [this] (let [rv (.hasNext src-iter)]
                                (when-not rv @closer*)
                                rv))
              (next [this] (.next src-iter))))))
      clojure.lang.IReduceInit
      (reduce [this rfn acc]
        (let [acc 
              (if (realized? seq*)
                (reduce rfn acc @seq*)
                (reduce rfn acc rv))]
          @closer*
          acc)))))
