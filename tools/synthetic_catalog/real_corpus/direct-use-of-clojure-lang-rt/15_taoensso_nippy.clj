(ns taoensso.nippy.io
  "Private low-level utils for reading/writing data, don't use."
  (:require
   [taoensso.truss  :as truss]
   [taoensso.encore :as enc]
   [taoensso.nippy
    [impl   :as impl]
    [schema :as sc]])

  (:import
   [taoensso.nippy.impl Cached CacheState SharedDict]
   [java.nio.charset StandardCharsets]
   [java.nio ByteBuffer BufferOverflowException]
   [java.io
    DataOutput       DataInput
    DataOutputStream DataInputStream
    ByteArrayOutputStream ByteArrayInputStream]))

(defn- bb-advance   [^ByteBuffer bb ^long n] (.position bb (+ (.position bb) n)) bb)

(defn- bb-readable! [^ByteBuffer bb ^long n] (when (> n (.remaining bb)) (throw (java.io.EOFException. (str "ByteBuffer underflow: need " n " bytes, have " (.remaining bb) ".")))))

(defn  bb-big-endian!
  ^ByteBuffer [^ByteBuffer bb]
  (when-not (= (.order bb) java.nio.ByteOrder/BIG_ENDIAN)
    (throw
      (IllegalArgumentException.
        (str "ByteBuffer must use BIG_ENDIAN order for DataInput/DataOutput semantics (have " (.order bb) ")."))))
  bb)

(definterface IByteReader
  ;; Common read interface implemented by both `DataInput` and `ByteBuffer`.
  ;; Uses `definterface` rather than `defprotocol` so the Clojure compiler
  ;; can emit typed invokevirtual/invokeinterface bytecode when the parameter
  ;; has ^IByteReader hint, letting the JIT devirtualize the hot read path.
  (^byte   readByte   [])
  (^short  readShort  [])
  (^int    readInt    [])
  (^long   readLong   [])
  (^float  readFloat  [])
  (^double readDouble [])
  (^char   readChar   [])
  (readFully     [^bytes ba ^int off ^int len])
  (skipBytes     [^int n])
  (toDataInput   [])
  (toInputStream [])
  (toByteBuffer  []))

(deftype ByteBufferReader [^ByteBuffer bb]
  IByteReader
  (readByte      [_] (.get       bb))
  (readShort     [_] (.getShort  bb))
  (readInt       [_] (.getInt    bb))
  (readLong      [_] (.getLong   bb))
  (readFloat     [_] (.getFloat  bb))
  (readDouble    [_] (.getDouble bb))
  (readChar      [_] (.getChar   bb))
  (readFully     [_ ^bytes ba ^int off ^int len] (.get bb ba off len))
  (skipBytes     [_ ^int n] (bb-advance bb n))
  (toDataInput   [_] (bb->din bb))
  (toByteBuffer  [_]          bb)
  (toInputStream [_]
    (proxy [java.io.InputStream] []
      (available [] (.remaining bb))
      (skip [n]
        (let [n (long (min (max (long n) 0) (.remaining bb)))]
          (bb-advance bb n)
          n))

      (read
        ([         ] (if (.hasRemaining bb) (bit-and 0xff (int (.get bb))) -1))
        ([^bytes ba]
         (let [len (alength ba)]
           (cond
             (zero? len)               0
             (not (.hasRemaining bb)) -1
             :else
             (let [n (min len (.remaining bb))]
               (.get bb ba 0 n)
               n))))

        ([^bytes ba off len]
         (let [ba-len (alength ba)]
           (cond
             (or (neg? off) (neg? len) (> off ba-len) (> len (- ba-len off)))
             (throw (IndexOutOfBoundsException.))

             (zero? len)               0
             (not (.hasRemaining bb)) -1
             :else
             (let [n (min len (.remaining bb))]
               (.get bb ba off n)
               n))))))))

(defmacro read-sm-ucount [ibr] `(- (int (.readByte  ~ibr)) Byte/MIN_VALUE))

(defmacro read-sm-count  [ibr]    `(int (.readByte  ~ibr)))

(defmacro read-md-count  [ibr]    `(int (.readShort ~ibr)))

(defmacro read-lg-count  [ibr]         `(.readInt   ~ibr))

(defn- ensure-readable-length!
  "Returns given length `n`, or throws `EOFException` if `n` items obviously
  cannot be read from `ibr`. Used to reject malformed length prefixes BEFORE
  they're used to size an allocation.

  `bytes-per-item` is a lower bound on each item's SERIALIZED size (1 for
  type-prefixed items, which always cost >= their type prefix).

  Note that only buffered input has a known remaining length. Stream input
  can be checked only for negative lengths, so a forged positive length may
  still trigger a large allocation there before the underlying stream is
  found to be short.

  Bounding stream input too was considered and intentionally rejected: it'd
  cost extra copying on large reads, and wouldn't make thaw resource-safe
  against hostile input anyway since decompression allocates upstream of
  this check. See commit message for details."
  ^long [^IByteReader ibr ^long n ^long bytes-per-item]
  (when (neg? n)
    (throw (java.io.EOFException. (str "Negative length: " n "."))))
  (when (instance? ByteBufferReader ibr)
    (let [required (* n bytes-per-item)
          remaining (.remaining ^ByteBuffer (.toByteBuffer ibr))]
      (when (> required remaining)
        (throw
          (java.io.EOFException.
            (str "ByteBuffer underflow: need at least " required
              " bytes, have " remaining "."))))))
  n)

(defn- ensure-non-negative-count!
  "Returns given count `n`, or throws `EOFException` if `n` is negative.

  Used on paths that build their result incrementally, so unlike
  `ensure-readable-length!` there's no allocation to guard here. The risk is
  instead that `enc/reduce-n` treats a negative count as zero, so a damaged
  count prefix would otherwise quietly yield an EMPTY coll rather than an error."
  ^long [^long n]
  (if (neg? n)
    (throw (java.io.EOFException. (str "Negative count: " n ".")))
    n))

(defn    read-bytes
  ([^IByteReader ibr len]
   (let [len (int (ensure-readable-length! ibr len 1))
         ba  (byte-array  len)]
     (.readFully ibr ba 0 len)
     ba))

  ([^IByteReader ibr]
   (enc/case-eval (int (.readByte ibr))
     sc/id-byte-array-0  (byte-array 0)
     sc/id-byte-array-sm (read-bytes ibr (read-sm-count ibr))
     sc/id-byte-array-md (read-bytes ibr (read-md-count ibr))
     sc/id-byte-array-lg (read-bytes ibr (read-lg-count ibr)))))

(defn read-str
  ([^IByteReader ibr len]
   (let [len (int len)]
     (if (instance? ByteBufferReader ibr)
       (let [^ByteBuffer bb (.toByteBuffer ibr)]
         (if (and (.hasArray bb) (<= 0 len (.remaining bb)))
           (let [pos (.position bb)
                 s   (String. ^bytes (.array bb) (+ (.arrayOffset bb) pos) len StandardCharsets/UTF_8)]
             (bb-advance bb len)
             s)
           (String. ^bytes (read-bytes ibr len) StandardCharsets/UTF_8)))
       (do (String. ^bytes (read-bytes ibr len) StandardCharsets/UTF_8)))))

  ([^IByteReader ibr]
   (enc/case-eval (int (.readByte ibr))
     sc/id-str-0   ""
     sc/id-str-sm* (read-str ibr (read-sm-ucount ibr))
     sc/id-str-sm_ (read-str ibr (read-sm-count  ibr))
     sc/id-str-md  (read-str ibr (read-md-count  ibr))
     sc/id-str-lg  (read-str ibr (read-lg-count  ibr)))))

(defn read-biginteger [^IByteReader ibr] (BigInteger. ^bytes (read-bytes ibr (.readInt ibr))))

(def ^:private ^:const kw-slash (int \/))

(defn- kw-from-bytes
  "Returns `Keyword` decoded from `len` UTF-8 bytes at `off`.
  Splits ns/name on the raw bytes so only the 2 final strings are
  allocated (vs 3 when going via (keyword <string>))."
  ^clojure.lang.Keyword [^bytes arr ^long off ^long len]
  (let [end (+ off len)
        slash
        (loop [idx off]
          (enc/cond
            (>= idx end)                       -1
            (== (int (aget arr idx)) kw-slash) idx
            :else (recur        (unchecked-inc idx))))]

    (if (or (== slash -1) (== len 1)) ; len 1 => the `:/` keyword
      (clojure.lang.Keyword/intern (String. arr (int off) (int len) StandardCharsets/UTF_8))
      (clojure.lang.Keyword/intern
        (String. arr (int  off)                  (int (- slash off))   StandardCharsets/UTF_8)
        (String. arr (int (unchecked-inc slash)) (int (- end slash 1)) StandardCharsets/UTF_8)))))

(defn read-kw
  "Reads a keyword of `len` UTF-8 bytes."
  [^IByteReader ibr ^long len]
  (if-not (instance? ByteBufferReader ibr)
    (keyword (read-str ibr len)) ; Stream input
    (let [^ByteBuffer bb (.toByteBuffer ibr)]
      (if-not (and (.hasArray bb) (<= 0 len (.remaining bb)))
        (keyword (read-str ibr len))
        (let [^bytes arr (.array bb)
              off (+ (.arrayOffset bb) (.position bb))]
          (bb-advance    bb len)
          (kw-from-bytes arr off len))))))

(defmacro ^:private read-dyn-array
  "Reads an array of individually type-prefixed elements."
  [ibr thaw-type array-type array]
  (let [thawed-sym (with-meta 'thawed-sym {:tag thaw-type})
        array-sym  (with-meta 'array-sym  {:tag array-type})]
    `(let [~array-sym ~array]
       (enc/reduce-n
         (fn [_# idx#]
           (let [~thawed-sym (read-typed ~ibr)]
             (aset ~'array-sym idx# ~'thawed-sym)))
         nil (alength ~'array-sym))
       ~'array-sym)))

(defmacro ^:private read-prim-array
  "Reads a primitive array of homogeneous elements."
  [ibr array-type array-fn as-buffer bytes read-el]
  (let [array-sym (with-meta 'array-sym {:tag array-type})]
    `(let [alen#      (read-lg-count ~ibr)
           _#         (ensure-readable-length! ~ibr alen# ~bytes)
           ~array-sym (~array-fn alen#)]
       (if (instance? ByteBufferReader ~ibr) ; Fast bulk read
         (let [^ByteBuffer bb# (.toByteBuffer ~ibr)]
           (.get (~as-buffer bb#) ~'array-sym)
           (bb-advance bb# (* alen# ~bytes)))
         (enc/reduce-n ; Element-wise read for legacy `DataInput` support
           (fn [_# idx#] (aset ~'array-sym idx# (~read-el ~ibr)))
           nil alen#))
       ~'array-sym)))

(defn read-vec [^IByteReader ibr ^long n]
  (if (or taoensso.nippy/*thaw-xform* (> n 32))
    (read-into [] ibr n) ; Checks count
    (let [items (object-array (ensure-non-negative-count! n))]
      (loop [idx 0]
        (when (< idx n)
          (aset items idx (read-typed ibr))
          (recur (unchecked-inc-int idx))))
      (clojure.lang.LazilyPersistentVector/createOwning items))))

(defn- read-list-via-array
  "Reads `n` type-prefixed elements into a `PersistentList` via a temporary
  array, avoiding the intermediate vector needed by `read-into`."
  [^IByteReader ibr ^long n]
  (let [items (object-array n)]
    (loop [idx 0]
      (when (< idx n)
        (aset items idx (read-typed ibr))
        (recur (unchecked-inc idx))))

    ;; Cons backwards so that element order matches the stream, and so that
    ;; the result is a `PersistentList` exactly as `(into () ...)` would give
    (loop [idx (unchecked-dec n), acc clojure.lang.PersistentList/EMPTY]
      (if (neg? idx)
        acc
        (recur (unchecked-dec idx)
          (.cons ^clojure.lang.IPersistentCollection acc (aget items idx)))))))

(defn read-list
  "Reads `n` type-prefixed elements as a `PersistentList`.

  `bounded-count?` should be true iff `n` came from a sm/md (byte/short)
  prefix, so is inherently too small to enable an allocation attack. Only
  then may we pre-size an array without first checking readable length."
  [^IByteReader ibr ^long n bounded-count?]
  (enc/cond
    ;; A transducer may filter, expand, carry state, and do completion work,
    ;; so the general (incremental) path is needed to apply it correctly
    taoensso.nippy/*thaw-xform*
    (into () (rseq (read-into [] ibr n)))

    bounded-count?
    (read-list-via-array ibr (ensure-non-negative-count! n))

    (instance? ByteBufferReader ibr)
    (read-list-via-array ibr (long (ensure-readable-length! ibr n 1)))

    ;; Stream input has no known remaining length, so build incrementally
    ;; rather than pre-sizing an array from an unverifiable count
    :else (into () (rseq (read-into [] ibr n)))))

(defn- read-kvs-via-rt-map [^IByteReader ibr ^long n]
  (let [kvs (object-array (* 2 n))]
    (enc/reduce-n
      (fn [_ idx]
        (let [offset (* 2 idx)]
          (aset kvs      offset  (read-typed ibr))
          (aset kvs (inc offset) (read-typed ibr))))
      nil n)

    (try
      (clojure.lang.RT/map kvs)
      (catch IllegalArgumentException _ ; Duplicate thawed keys: preserve assoc semantics
        (persistent!
          (enc/reduce-n
            (fn [m idx]
              (let [offset (* 2 idx)]
                (assoc! m (aget kvs offset) (aget kvs (inc offset)))))
            (transient {}) n))))))

(defn read-map [^IByteReader ibr ^long n]
  ;; Bound buffering to Clojure v1.13's maximum possible PAM size. The existing
  ;; paths remain faster for tiny maps and avoid temporary arrays for large maps.
  (if (or taoensso.nippy/*thaw-xform* (<= n 10) (> n 64))
    (read-kvs-into {}    ibr n)
    (read-kvs-via-rt-map ibr n)))

(defn read-kvs-depr [to ^IByteReader ibr] (read-kvs-into to ibr (quot (.readInt ibr) 2)))

(defn read-custom [^IByteReader ibr prefixed? type-id]
  (if-let [custom-reader (get taoensso.nippy/*custom-readers* type-id)]
    (try
      (custom-reader (.toDataInput ibr))
      (catch Exception e
        (truss/ex-info!
          (str "Reader exception for custom type id: " type-id)
          {:type-id type-id, :prefixed? prefixed?} e)))

    (truss/ex-info!
      (str "No reader provided for custom type id: " type-id)
      {:type-id type-id, :prefixed? prefixed?})))

(defn read-sz!!
  "Reads object using Java `Serializable`. May be unsafe!"
  [^java.io.InputStream input-stream class-name]
  (try
    (let [obj (.readObject (java.io.ObjectInputStream. input-stream))] ; May be unsafe!
      (try
        (cast (Class/forName class-name) obj)
        (catch Exception e
          {:nippy/unthawable
           {:type  :serializable
            :cause :exception

            :class-name class-name
            :content    obj
            :exception  e}})))

    (catch Exception e
      {:nippy/unthawable
       {:type  :serializable
        :cause :exception

        :class-name class-name
        :content    nil
        :exception  e}})))

(defn read-sz [^IByteReader ibr class-name legacy?]
  (if legacy?

    ;; Serialized object directly to stream WITHOUT length prefix
    (if (impl/thaw-serializable-allowed? class-name)
      (read-sz!! (.toInputStream ibr)    class-name)
      (truss/ex-info! ; No way to skip bytes, so best we can do is throw
        "Cannot thaw object: `taoensso.nippy/*thaw-serializable-allowlist*` check failed. This is a security feature. See `*thaw-serializable-allowlist*` docstring or https://github.com/ptaoussanis/nippy/issues/130 for details!"
        {:class-name class-name}))

    (let [sz-ba (read-bytes ibr)]
      (if (impl/thaw-serializable-allowed?       class-name)
        (read-sz!! (ByteArrayInputStream. sz-ba) class-name)
        {:nippy/unthawable
         {:type  :serializable
          :cause :quarantined

          :class-name class-name
          :content    sz-ba}}))))

(defn- read-deftype-fields ^objects [^IByteReader ibr n]
  (let [vals (object-array n)]
    (enc/reduce-n (fn [_ i] (aset vals i (read-typed ibr))) nil n)
    vals))

(defn read-deftype [^IByteReader ibr class-name legacy?]
  (if legacy? ; No field count in payload
    (try
      (let [c   (clojure.lang.RT/classForName class-name)
            ctr (aget (.getConstructors c) 0)
            num-fields-exp (count (impl/get-basis-fields c))]

        (.newInstance ^java.lang.reflect.Constructor ctr
          (read-deftype-fields ibr num-fields-exp)))

      (catch Exception e
        {:nippy/unthawable
         {:type       :deftype
          :class-name class-name
          :cause      :exception
          :exception  e}}))

    (let [num-fields (read-sm-ucount      ibr)
          field-vals (read-deftype-fields ibr num-fields) ; Always read all fields first
          ]
      (try
        (let [c   (clojure.lang.RT/classForName class-name)
              ctr (aget (.getConstructors c) 0)
              num-fields-exp (count (impl/get-basis-fields c))]

          (if (== num-fields num-fields-exp)
            (.newInstance ^java.lang.reflect.Constructor ctr field-vals)
            {:nippy/unthawable
             {:type       :deftype
              :class-name class-name
              :cause      :field-num-mismatch
              :field-num  {:expected num-fields-exp, :actual num-fields}
              :content    (vec field-vals)}}))

        (catch Exception e ; e.g. class not found, constructor failure
          {:nippy/unthawable
           {:type       :deftype
            :class-name class-name
            :cause      :exception
            :content    (vec field-vals)
            :exception  e}})))))

(defn read-typed
  "Reads one object as type-prefixed bytes from given `IByteReader`."
  [^IByteReader  ibr]
  (let [type-id (int (.readByte ibr))]
    (impl/when-debug (println (str "read-typed: " type-id)))
    (try
      (enc/case-eval type-id

        sc/id-nil               nil
        sc/id-true              true
        sc/id-false             false
        sc/id-meta-protocol-key impl/meta-protocol-key

        sc/id-reader-sm  (impl/read-edn   (read-str ibr (read-sm-count ibr)))
        sc/id-reader-md  (impl/read-edn   (read-str ibr (read-md-count ibr)))
        sc/id-reader-lg  (impl/read-edn   (read-str ibr (read-lg-count ibr)))
        sc/id-reader-lg_ (impl/read-edn   (read-str ibr (read-lg-count ibr)))
        sc/id-record-sm  (read-record ibr (read-str ibr (read-sm-count ibr)))
        sc/id-record-md  (read-record ibr (read-str ibr (read-md-count ibr)))
        sc/id-record-lg_ (read-record ibr (read-str ibr (read-lg-count ibr)))

        sc/id-sz-sm  (read-sz ibr (read-str ibr (read-sm-count ibr)) false)
        sc/id-sz-md  (read-sz ibr (read-str ibr (read-md-count ibr)) false)
        sc/id-sz-sm_ (read-sz ibr (read-str ibr (read-sm-count ibr)) :legacy)
        sc/id-sz-md_ (read-sz ibr (read-str ibr (read-md-count ibr)) :legacy)
        sc/id-sz-lg_ (read-sz ibr (read-str ibr (read-lg-count ibr)) :legacy)

        sc/id-deftype  (read-deftype ibr (read-typed ibr) false)
        sc/id-deftype_ (read-deftype ibr (read-typed ibr) :legacy)
        sc/id-char     (.readChar    ibr)

        sc/id-meta
        (let [m (read-typed ibr) ; Always consume from stream
              x (read-typed ibr)]
          (if-let [m (when taoensso.nippy/*incl-metadata?* (not-empty (dissoc m impl/meta-protocol-key)))]
            (with-meta x m)
            (do        x)))

        sc/id-cached-0  (impl/read-cached read-typed 0 ibr)
        sc/id-cached-1  (impl/read-cached read-typed 1 ibr)
        sc/id-cached-2  (impl/read-cached read-typed 2 ibr)
        sc/id-cached-3  (impl/read-cached read-typed 3 ibr)
        sc/id-cached-4  (impl/read-cached read-typed 4 ibr)
        sc/id-cached-5  (impl/read-cached read-typed 5 ibr)
        sc/id-cached-6  (impl/read-cached read-typed 6 ibr)
        sc/id-cached-7  (impl/read-cached read-typed 7 ibr)
        sc/id-cached-sm (impl/read-cached read-typed (read-sm-count ibr) ibr)
        sc/id-cached-md (impl/read-cached read-typed (read-md-count ibr) ibr)

        sc/id-shared-dict
        (do
          (impl/read-shared-dict-marker! (read-md-count ibr) (.readLong ibr))
          (read-typed ibr))

        sc/id-byte-array-0    (byte-array 0)
        sc/id-byte-array-sm   (read-bytes ibr (read-sm-count ibr))
        sc/id-byte-array-md   (read-bytes ibr (read-md-count ibr))
        sc/id-byte-array-lg   (read-bytes ibr (read-lg-count ibr))

        sc/id-string-array-lg  (read-dyn-array ibr String "[Ljava.lang.String;" (make-array String (ensure-readable-length! ibr (read-lg-count ibr) 1)))
        sc/id-object-array-lg  (read-dyn-array ibr Object "[Ljava.lang.Object;" (object-array      (ensure-readable-length! ibr (read-lg-count ibr) 1)))
        sc/id-int-array-lg_    (read-dyn-array ibr int    "[I"                  (int-array         (ensure-readable-length! ibr (read-lg-count ibr) 1)))
        sc/id-long-array-lg_   (read-dyn-array ibr long   "[J"                  (long-array        (ensure-readable-length! ibr (read-lg-count ibr) 1)))
        sc/id-float-array-lg_  (read-dyn-array ibr float  "[F"                  (float-array       (ensure-readable-length! ibr (read-lg-count ibr) 1)))
        sc/id-double-array-lg_ (read-dyn-array ibr double "[D"                  (double-array      (ensure-readable-length! ibr (read-lg-count ibr) 1)))

        sc/id-int-array-lg     (read-prim-array ibr "[I" int-array    .asIntBuffer    Integer/BYTES .readInt)
        sc/id-long-array-lg    (read-prim-array ibr "[J" long-array   .asLongBuffer      Long/BYTES .readLong)
        sc/id-float-array-lg   (read-prim-array ibr "[F" float-array  .asFloatBuffer    Float/BYTES .readFloat)
        sc/id-double-array-lg  (read-prim-array ibr "[D" double-array .asDoubleBuffer  Double/BYTES .readDouble)

        sc/id-str-0       ""
        sc/id-str-sm*              (read-str ibr (read-sm-ucount ibr))
        sc/id-str-sm_              (read-str ibr (read-sm-count  ibr))
        sc/id-str-md               (read-str ibr (read-md-count  ibr))
        sc/id-str-lg               (read-str ibr (read-lg-count  ibr))

        sc/id-kw-sm       (read-kw ibr (read-sm-count ibr))
        sc/id-kw-md       (read-kw ibr (read-md-count ibr))
        sc/id-kw-md_      (read-kw ibr (read-lg-count ibr))
        sc/id-kw-lg_      (read-kw ibr (read-lg-count ibr))

        sc/id-sym-sm      (symbol  (read-str ibr (read-sm-count ibr)))
        sc/id-sym-md      (symbol  (read-str ibr (read-md-count ibr)))
        sc/id-sym-md_     (symbol  (read-str ibr (read-lg-count ibr)))
        sc/id-sym-lg_     (symbol  (read-str ibr (read-lg-count ibr)))
        sc/id-regex       (re-pattern            (read-typed    ibr))

        sc/id-vec-0       []
        sc/id-vec-2       (read-vec ibr 2)
        sc/id-vec-3       (read-vec ibr 3)
        sc/id-vec-sm*     (read-vec ibr (read-sm-ucount ibr))
        sc/id-vec-sm_     (read-vec ibr (read-sm-count  ibr))
        sc/id-vec-md      (read-vec ibr (read-md-count  ibr))
        sc/id-vec-lg      (read-vec ibr (read-lg-count  ibr))

        sc/id-set-0       #{}
        sc/id-set-sm*     (read-into    #{} ibr (read-sm-ucount ibr))
        sc/id-set-sm_     (read-into    #{} ibr (read-sm-count  ibr))
        sc/id-set-md      (read-into    #{} ibr (read-md-count  ibr))
        sc/id-set-lg      (read-into    #{} ibr (read-lg-count  ibr))

        sc/id-map-0       {}
        sc/id-map-sm*     (read-map ibr (read-sm-ucount ibr))
        sc/id-map-sm_     (read-map ibr (read-sm-count  ibr))
        sc/id-map-md      (read-map ibr (read-md-count  ibr))
        sc/id-map-lg      (read-map ibr (read-lg-count  ibr))
        sc/id-pam-sm*_    (read-map ibr (read-sm-ucount ibr)) ; Retired encoding, see schema

        sc/id-queue-lg      (read-into     clojure.lang.PersistentQueue/EMPTY ibr (read-lg-count ibr))
        sc/id-sorted-set-lg (read-into     (sorted-set)                       ibr (read-lg-count ibr))
        sc/id-sorted-map-lg (read-kvs-into (sorted-map)                       ibr (read-lg-count ibr))

        sc/id-list-0            ()
        sc/id-list-sm     (read-list ibr (read-sm-count ibr) true)
        sc/id-list-md     (read-list ibr (read-md-count ibr) true)
        sc/id-list-lg     (read-list ibr (read-lg-count ibr) false)

        sc/id-seq-0       (lazy-seq nil)
        sc/id-seq-sm      (or (seq (read-into [] ibr (read-sm-count ibr))) (lazy-seq nil))
        sc/id-seq-md      (or (seq (read-into [] ibr (read-md-count ibr))) (lazy-seq nil))
        sc/id-seq-lg      (or (seq (read-into [] ibr (read-lg-count ibr))) (lazy-seq nil))

        sc/id-byte              (.readByte  ibr)
        sc/id-short             (.readShort ibr)
        sc/id-integer           (.readInt   ibr)
        sc/id-long-0      0
        sc/id-long-sm_    (long (.readByte  ibr))
        sc/id-long-md_    (long (.readShort ibr))
        sc/id-long-lg_    (long (.readInt   ibr))
        sc/id-long-xl           (.readLong  ibr)

        sc/id-long-pos-sm    (- (long (.readByte  ibr))    Byte/MIN_VALUE)
        sc/id-long-pos-md    (- (long (.readShort ibr))   Short/MIN_VALUE)
        sc/id-long-pos-lg    (- (long (.readInt   ibr)) Integer/MIN_VALUE)

        sc/id-long-neg-sm (- (- (long (.readByte  ibr))    Byte/MIN_VALUE))
        sc/id-long-neg-md (- (- (long (.readShort ibr))   Short/MIN_VALUE))
        sc/id-long-neg-lg (- (- (long (.readInt   ibr)) Integer/MIN_VALUE))

        sc/id-bigint      (bigint (read-biginteger ibr))
        sc/id-biginteger          (read-biginteger ibr)

        sc/id-float       (.readFloat  ibr)
        sc/id-double-0    0.0
        sc/id-double      (.readDouble ibr)

        sc/id-bigdec      (BigDecimal. ^BigInteger (read-biginteger ibr) (.readInt        ibr))
        sc/id-ratio       (clojure.lang.Ratio.     (read-biginteger ibr) (read-biginteger ibr))

        sc/id-map-entry   (enc/map-entry (read-typed ibr) (read-typed ibr))

        sc/id-util-date   (java.util.Date. (.readLong ibr))
        sc/id-sql-date    (java.sql.Date.  (.readLong ibr))
        sc/id-uuid        (java.util.UUID. (.readLong ibr) (.readLong ibr))
        sc/id-uri         (java.net.URI.   (read-typed ibr))

        sc/id-prefixed-custom-md (read-custom ibr :prefixed (.readShort ibr))

        sc/id-time-instant
        (let [secs  (.readLong ibr)
              nanos (.readInt  ibr)]

          (enc/compile-if java.time.Instant
            (java.time.Instant/ofEpochSecond secs nanos)
            {:nippy/unthawable
             {:type  :class
              :cause :class-not-found
              :class-name "java.time.Instant"
              :content    {:epoch-second secs :nano nanos}}}))

        sc/id-time-duration
        (let [secs  (.readLong ibr)
              nanos (.readInt  ibr)]

          (enc/compile-if java.time.Duration
            (java.time.Duration/ofSeconds secs nanos)
            {:nippy/unthawable
             {:type       :class
              :cause      :class-not-found
              :class-name "java.time.Duration"
              :content    {:seconds secs :nanos nanos}}}))

        sc/id-time-period
        (let [years  (.readInt ibr)
              months (.readInt ibr)
              days   (.readInt ibr)]

          (enc/compile-if java.time.Period
            (java.time.Period/of years months days)
            {:nippy/unthawable
             {:type       :class
              :cause      :class-not-found
              :class-name "java.time.Period"
              :content    {:years years :months months :days days}}}))

        ;; Deprecated ------------------------------------------------------
        sc/id-boolean_    (not (zero? (int (.readByte ibr))))
        sc/id-sorted-map_ (read-kvs-depr (sorted-map) ibr)
        sc/id-map__       (read-kvs-depr {} ibr)
        sc/id-reader_     (impl/read-edn (.readUTF ^DataInput (.toDataInput ibr)))
        sc/id-str_                       (.readUTF ^DataInput (.toDataInput ibr))
        sc/id-kw_               (keyword (.readUTF ^DataInput (.toDataInput ibr)))
        sc/id-map_
        (apply hash-map
          (enc/repeatedly-into [] (* 2 (.readInt ibr))
            (fn [] (read-typed ibr))))
        ;; -----------------------------------------------------------------

        ;; else
        (if (neg? type-id)
          (read-custom ibr false type-id) ; Unprefixed custom type
          (truss/ex-info!
            (str "Unrecognized type id (" type-id "). Data frozen with newer Nippy version?")
            {:type-id type-id})))

      (catch Throwable t
        (truss/ex-info! (str "Thaw failed against type-id: " type-id)
          {:type-id type-id} t)))))

(defn bb->din
  "Returns a `DataInput` adapter over given `ByteBuffer`.
  Reads from the buffer's current position and advances it."
  ^DataInput  [^ByteBuffer bb]
  (let [bb (bb-big-endian! bb)]
    (reify DataInput
      (^void readFully [_ ^bytes ba]
       (let [len (alength ba)]
         (bb-readable! bb len)
         (.get bb ba 0 len)
         nil))

      (^void readFully [_ ^bytes ba ^int off ^int len]
       (bb-readable! bb len)
       (.get bb ba off len)
       nil)

      (^int skipBytes [_ ^int n]
       (let [n       (max n 0)
             skipped (min n (.remaining bb))]
         (bb-advance bb skipped)
         (do            skipped)))

      (^boolean   readBoolean [_] (bb-readable! bb 1) (not (zero?   (int (.get bb)))))
      (^byte         readByte [_] (bb-readable! bb 1)                    (.get bb))
      (^int  readUnsignedByte [_] (bb-readable! bb 1) (bit-and 0xFF (int (.get bb))))

      (^short       readShort [_] (bb-readable! bb 2)                      (.getShort bb))
      (^int readUnsignedShort [_] (bb-readable! bb 2) (bit-and 0xFFFF (int (.getShort bb))))

      (^char     readChar [_] (bb-readable! bb 2) (.getChar   bb))
      (^int       readInt [_] (bb-readable! bb 4) (.getInt    bb))
      (^long     readLong [_] (bb-readable! bb 8) (.getLong   bb))
      (^float   readFloat [_] (bb-readable! bb 4) (.getFloat  bb))
      (^double readDouble [_] (bb-readable! bb 8) (.getDouble bb))

      (^String readUTF  [this] (DataInputStream/readUTF this))
      (^String readLine [_]
       (when (.hasRemaining bb)
         (let [sb (StringBuilder.)]
           (loop []
             (if-not (.hasRemaining bb)
               (.toString sb)
               (let   [b (bit-and 0xFF (int (.get bb)))]
                 (case b
                   10 (.toString sb)
                   13
                   (do
                     (when (and (.hasRemaining bb) (= 10 (bit-and 0xFF (int (.get bb (.position bb))))))
                       (bb-advance bb 1))
                     (.toString sb))

                   ;; else
                   (do (.append sb (char b)) (recur))))))))))))
