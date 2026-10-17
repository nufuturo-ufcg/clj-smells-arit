(ns coffi.mem
  "Functions for managing native allocations, memory arenas, and (de)serialization.

  For any new type to be implemented, three multimethods must be overriden, but
  which three depends on the native representation of the type.

  If the native representation of the type is a primitive (whether or not other
  data beyond the primitive is associated with it, as e.g. a pointer),
  then [[primitive-type]] must be overriden to return which primitive type it is
  serialized as, then [[serialize*]] and [[deserialize*]] should be overriden.

  If the native representation of the type is a composite type, like a union,
  struct, or array, then [[c-layout]] must be overriden to return the native
  layout of the type, and [[serialize-into]] and [[deserialize-from]] should be
  overriden to allow marshaling values of the type into and out of memory
  segments."
  (:require
   [clojure.pprint :as pprint]
   [clojure.set :as set]
   [clojure.spec.alpha :as s])
  (:import
   (java.lang.foreign
    AddressLayout
    Arena
    MemoryLayout
    MemoryLayout$PathElement
    MemorySegment
    MemorySegment$Scope
    SegmentAllocator
    ValueLayout
    ValueLayout$OfByte
    ValueLayout$OfShort
    ValueLayout$OfInt
    ValueLayout$OfLong
    ValueLayout$OfChar
    ValueLayout$OfFloat
    ValueLayout$OfDouble)
   (java.lang.ref Cleaner)
   (java.util.function Consumer)
   (java.nio ByteOrder))
  (:refer-clojure :exclude [defstruct]))

(defn auto-arena
  "Constructs a new memory arena that is managed by the garbage collector.

  The arena may be shared across threads, and all resources created with it will
  be cleaned up at the same time, when all references have been collected.

  This type of arena cannot be closed, and therefore should not be created in
  a [[with-open]] clause."
  ^Arena []
  (Arena/ofAuto))

(defn reinterpret
  "Reinterprets the `segment` as having the passed `size`.

  If `arena` is passed, the scope of the `segment` is associated with the arena,
  as well as its access constraints. If `cleanup` is passed, it will be a
  1-argument function of a fresh memory segment backed by the same memory as the
  returned segment which should perform any required cleanup operations. It will
  be called when the `arena` is closed."
  (^MemorySegment [^MemorySegment segment size]
   (.reinterpret segment (long size) (auto-arena) nil))
  (^MemorySegment [^MemorySegment segment size ^Arena arena]
   (.reinterpret segment (long size) arena nil))
  (^MemorySegment [^MemorySegment segment size ^Arena arena cleanup]
   (.reinterpret segment (long size) arena
                 (reify Consumer
                   (accept [_this segment]
                     (cleanup segment))))))

(def ^ByteOrder big-endian
  "The big-endian [[ByteOrder]].

  See [[little-endian]], [[native-endian]]."
  ByteOrder/BIG_ENDIAN)

(def ^ByteOrder little-endian
  "The little-endian [[ByteOrder]].

  See [[big-endian]], [[native-endian]]"
  ByteOrder/LITTLE_ENDIAN)

(def ^ByteOrder native-endian
  "The [[ByteOrder]] for the native endianness of the current hardware.

  See [[big-endian]], [[little-endian]]."
  (ByteOrder/nativeOrder))

(def ^ValueLayout$OfByte byte-layout
  "The [[MemoryLayout]] for a byte in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_BYTE)

(def ^ValueLayout$OfShort short-layout
  "The [[MemoryLayout]] for a c-sized short in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_SHORT)

(def ^ValueLayout$OfInt int-layout
  "The [[MemoryLayout]] for a c-sized int in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_INT)

(def ^ValueLayout$OfLong long-layout
  "The [[MemoryLayout]] for a c-sized long in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_LONG)

(def ^ValueLayout$OfByte char-layout
  "The [[MemoryLayout]] for a c-sized char in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_BYTE)

(def ^ValueLayout$OfFloat float-layout
  "The [[MemoryLayout]] for a c-sized float in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_FLOAT)

(def ^ValueLayout$OfDouble double-layout
  "The [[MemoryLayout]] for a c-sized double in [[native-endian]] [[ByteOrder]]."
  ValueLayout/JAVA_DOUBLE)

(def ^AddressLayout pointer-layout
  "The [[MemoryLayout]] for a native pointer in [[native-endian]] [[ByteOrder]]."
  ValueLayout/ADDRESS)

(defn read-byte
  "Reads a [[byte]] from the `segment`, at an optional `offset`."
  {:inline
   (fn read-byte-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfByte byte-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfByte byte-layout offset#))))}
  ([^MemorySegment segment]
   (.get segment ^ValueLayout$OfByte byte-layout 0))
  ([^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfByte byte-layout offset)))

(defn read-short
  "Reads a [[short]] from the `segment`, at an optional `offset`.

  If `byte-order` is not provided, it defaults to [[native-endian]]."
  {:inline
   (fn read-short-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfShort short-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfShort short-layout offset#)))
     ([segment offset byte-order]
      `(let [segment# ~segment
             offset# ~offset
             byte-order# ~byte-order]
         (.get ^MemorySegment segment# (.withOrder ^ValueLayout$OfShort short-layout ^ByteOrder byte-order#) offset#))))}
  ([^MemorySegment segment]
   (.get segment ^ValueLayout$OfShort short-layout 0))
  ([^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfShort short-layout offset))
  ([^MemorySegment segment ^long offset ^ByteOrder byte-order]
   (.get segment (.withOrder ^ValueLayout$OfShort short-layout byte-order) offset)))

(defn read-int
  "Reads a [[int]] from the `segment`, at an optional `offset`.

  If `byte-order` is not provided, it defaults to [[native-endian]]."
  {:inline
   (fn read-int-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfInt int-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfInt int-layout offset#)))
     ([segment offset byte-order]
      `(let [segment# ~segment
             offset# ~offset
             byte-order# ~byte-order]
         (.get ^MemorySegment segment# (.withOrder ^ValueLayout$OfInt int-layout ^ByteOrder byte-order#) offset#))))}
  ([^MemorySegment segment]
   (.get segment ^ValueLayout$OfInt int-layout 0))
  ([^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfInt int-layout offset))
  ([^MemorySegment segment ^long offset ^ByteOrder byte-order]
   (.get segment (.withOrder ^ValueLayout$OfInt int-layout byte-order) offset)))

(defn read-long
  "Reads a [[long]] from the `segment`, at an optional `offset`.

  If `byte-order` is not provided, it defaults to [[native-endian]]."
  {:inline
   (fn read-long-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfLong long-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfLong long-layout offset#)))
     ([segment offset byte-order]
      `(let [segment# ~segment
             offset# ~offset
             byte-order# ~byte-order]
         (.get ^MemorySegment segment# (.withOrder ^ValueLayout$OfLong long-layout ^ByteOrder byte-order#) offset#))))}
  (^long [^MemorySegment segment]
   (.get segment ^ValueLayout$OfLong long-layout 0))
  (^long [^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfLong long-layout offset))
  (^long [^MemorySegment segment ^long offset ^ByteOrder byte-order]
   (.get segment (.withOrder ^ValueLayout$OfLong long-layout byte-order) offset)))

(defn read-char
  "Reads a [[char]] from the `segment`, at an optional `offset`."
  {:inline
   (fn read-char-inline
     ([segment]
      `(let [segment# ~segment]
         (char (Byte/toUnsignedInt (.get ^MemorySegment segment# ^ValueLayout$OfByte byte-layout 0)))))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (char (Byte/toUnsignedInt (.get ^MemorySegment segment# ^ValueLayout$OfByte byte-layout offset#))))))}
  ([^MemorySegment segment]
   (char (Byte/toUnsignedInt (.get segment ^ValueLayout$OfChar byte-layout 0))))
  ([^MemorySegment segment ^long offset]
   (char (Byte/toUnsignedInt (.get segment ^ValueLayout$OfChar byte-layout offset)))))

(defn read-float
  "Reads a [[float]] from the `segment`, at an optional `offset`.

  If `byte-order` is not provided, it defaults to [[native-endian]]."
  {:inline
   (fn read-float-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfFloat float-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfFloat float-layout offset#)))
     ([segment offset byte-order]
      `(let [segment# ~segment
             offset# ~offset
             byte-order# ~byte-order]
         (.get ^MemorySegment segment# (.withOrder ^ValueLayout$OfFloat float-layout ^ByteOrder byte-order#) offset#))))}
  ([^MemorySegment segment]
   (.get segment ^ValueLayout$OfFloat float-layout 0))
  ([^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfFloat float-layout offset))
  ([^MemorySegment segment ^long offset ^ByteOrder byte-order]
   (.get segment (.withOrder ^ValueLayout$OfFloat float-layout byte-order) offset)))

(defn read-double
  "Reads a [[double]] from the `segment`, at an optional `offset`.

  If `byte-order` is not provided, it defaults to [[native-endian]]."
  {:inline
   (fn read-double-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^ValueLayout$OfDouble double-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^ValueLayout$OfDouble double-layout offset#)))
     ([segment offset byte-order]
      `(let [segment# ~segment
             offset# ~offset
             byte-order# ~byte-order]
         (.get ^MemorySegment segment# (.withOrder ^ValueLayout$OfDouble double-layout ^ByteOrder byte-order#) offset#))))}
  (^double [^MemorySegment segment]
   (.get segment ^ValueLayout$OfDouble double-layout 0))
  (^double [^MemorySegment segment ^long offset]
   (.get segment ^ValueLayout$OfDouble double-layout offset))
  (^double [^MemorySegment segment ^long offset ^ByteOrder byte-order]
   (.get segment (.withOrder ^ValueLayout$OfDouble double-layout byte-order) offset)))

(defn read-address
  "Reads an address from the `segment`, at an optional `offset`, wrapped in a [[MemorySegment]]."
  {:inline
   (fn read-address-inline
     ([segment]
      `(let [segment# ~segment]
         (.get ^MemorySegment segment# ^AddressLayout pointer-layout 0)))
     ([segment offset]
      `(let [segment# ~segment
             offset# ~offset]
         (.get ^MemorySegment segment# ^AddressLayout pointer-layout offset#))))}
  (^MemorySegment [^MemorySegment segment]
   (.get segment ^AddressLayout pointer-layout 0))
  (^MemorySegment [^MemorySegment segment ^long offset]
   (.get segment ^AddressLayout pointer-layout offset)))

(defn read-bytes
  "reads `n` elements from a `segment` to a [[byte]] array, at an optional `offset`."
  {:inline
   (fn read-bytes-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (byte-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfByte byte-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (byte-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfByte byte-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (byte-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfByte byte-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (byte-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfByte byte-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (byte-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfByte byte-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (byte-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfByte byte-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-shorts
  "reads `n` elements from a `segment` to a [[short]] array, at an optional `offset`."
  {:inline
   (fn read-shorts-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (short-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfShort short-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (short-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfShort short-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (short-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfShort short-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (short-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfShort short-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (short-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfShort short-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (short-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfShort short-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-longs
  "reads `n` elements from a `segment` to a [[long]] array, at an optional `offset`."
  {:inline
   (fn read-longs-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (long-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfLong long-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (long-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfLong long-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (long-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfLong long-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (long-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfLong long-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (long-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfLong long-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (long-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfLong long-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-ints
  "reads `n` elements from a `segment` to a [[int]] array, at an optional `offset`."
  {:inline
   (fn read-ints-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (int-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfInt int-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (int-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfInt int-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (int-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfInt int-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (int-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfInt int-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (int-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfInt int-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (int-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfInt int-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-chars
  "reads `n` elements from a `segment` to a [[char]] array, at an optional `offset`."
  {:inline
   (fn read-chars-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (char-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfChar char-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (char-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfChar char-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (char-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfChar char-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (char-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfChar char-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (char-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfChar char-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (char-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfChar char-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-floats
  "reads `n` elements from a `segment` to a [[float]] array, at an optional `offset`."
  {:inline
   (fn read-floats-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (float-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfFloat float-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (float-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfFloat float-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (float-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfFloat float-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (float-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfFloat float-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (float-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfFloat float-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (float-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfFloat float-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn read-doubles
  "reads `n` elements from a `segment` to a [[double]] array, at an optional `offset`."
  {:inline
   (fn read-doubles-inline
     ([segment n]
      `(let [n# ~n
             segment# ~segment
             arr# (double-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfDouble double-layout 0 arr# 0 n#)
         arr#))
     ([segment n offset]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             arr# (double-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# ^ValueLayout$OfDouble double-layout offset# arr# 0 n#)
         arr#))
     ([segment n offset byte-order]
      `(let [n# ~n
             segment# ~segment
             offset# ~offset
             byte-order# ~byte-order
             arr# (double-array ~n)]
         (MemorySegment/copy ^MemorySegment segment# (.withOrder ^ValueLayout$OfDouble double-layout ^ByteOrder byte-order#) offset# arr# 0 n#)
         arr#)))}
  ([^MemorySegment segment n]
   (let [arr (double-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfDouble double-layout 0 arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset]
   (let [arr (double-array n)]
     (MemorySegment/copy segment ^ValueLayout$OfDouble double-layout offset arr 0 ^int n)
     arr))
  ([^MemorySegment segment n ^long offset ^ByteOrder byte-order]
   (let [arr (double-array n)]
     (MemorySegment/copy segment (.withOrder ^ValueLayout$OfDouble double-layout byte-order) offset arr 0 ^int n)
     arr)))

(defn- type-dispatch
  "Gets a type dispatch value from a (potentially composite) type."
  [type]
  (cond
    (qualified-keyword? type) type
    (sequential? type) (keyword (first type))
    :else (throw (ex-info "Invalid type object" {:type type}))))

(defmulti primitive-type
  "Gets the primitive type that is used to pass as an argument for the `type`.

  This is for objects which are passed to native functions as primitive types,
  but which need additional logic to be performed during serialization and
  deserialization.

  Implementations of this method should take into account that type arguments
  may not always be evaluated before passing to this function.

  Returns nil for any type which does not have a primitive representation."
  type-dispatch)

(defmethod primitive-type :default
  [_type]
  nil)

(defmethod primitive-type ::byte
  [_type]
  ::byte)

(defmethod primitive-type ::short
  [_type]
  ::short)

(defmethod primitive-type ::int
  [_type]
  ::int)

(defmethod primitive-type ::long
  [_type]
  ::long)

(defmethod primitive-type ::char
  [_type]
  ::char)

(defmethod primitive-type ::float
  [_type]
  ::float)

(defmethod primitive-type ::double
  [_type]
  ::double)

(defmethod primitive-type ::pointer
  [_type]
  ::pointer)

(defmethod primitive-type ::void
  [_type]
  ::void)

(defmulti c-layout
  "Gets the layout object for a given `type`.

  If a type is primitive it will return the appropriate primitive
  layout (see [[c-prim-layout]]).

  Otherwise, it should return a [[GroupLayout]] for the given type."
  type-dispatch)

(defmethod c-layout :default
  [type]
  (c-layout (primitive-type type)))

(defmethod c-layout ::byte
  [_type]
  byte-layout)

(defmethod c-layout ::short
  [type]
  (if (sequential? type)
    (.withOrder short-layout ^ByteOrder (second type))
    short-layout))

(defmethod c-layout ::int
  [type]
  (if (sequential? type)
    (.withOrder int-layout ^ByteOrder (second type))
    int-layout))

(defmethod c-layout ::long
  [type]
  (if (sequential? type)
    (.withOrder long-layout ^ByteOrder (second type))
    long-layout))

(defmethod c-layout ::char
  [_type]
  char-layout)

(defmethod c-layout ::float
  [type]
  (if (sequential? type)
    (.withOrder float-layout ^ByteOrder (second type))
    float-layout))

(defmethod c-layout ::double
  [type]
  (if (sequential? type)
    (.withOrder double-layout ^ByteOrder (second type))
    double-layout))

(defmethod c-layout ::pointer
  [_type]
  pointer-layout)

(defn size-of
  "The size in bytes of the given `type`."
  ^long [type]
  (let [t (cond-> type
            (not (instance? MemoryLayout type)) c-layout)]
    (.byteSize ^MemoryLayout t)))

(defmethod c-layout ::raw
  [[_raw type]]
  (c-layout type))

(defmethod primitive-type ::c-string
  [_type]
  ::pointer)

(defmethod c-layout ::union
  [[_union types & {:as _opts} :as _type]]
  (let [items (if (map? types)
                (map
                  (fn [[field-name field]]
                    (.withName ^MemoryLayout (c-layout field) (name field-name)))
                  types)
                (map c-layout types))]
    (MemoryLayout/unionLayout
     (into-array MemoryLayout items))))

(defmethod c-layout ::struct
  [[_struct fields]]
  (let [fields (for [[field-name field] fields]
                 (.withName ^MemoryLayout (c-layout field)
                            (name field-name)))]
    (MemoryLayout/structLayout
     (into-array MemoryLayout fields))))

(defmethod c-layout ::padding
  [[_padding size]]
  (MemoryLayout/paddingLayout size))

(defmethod c-layout ::array
  [[_array type count]]
  (MemoryLayout/sequenceLayout
   count
   (c-layout type)))

(defmethod primitive-type ::enum
  [[_enum _variants & {:keys [repr]}]]
  (if repr
    (primitive-type repr)
    ::int))

(defmethod primitive-type ::flagset
  [[_flagset _bits & {:keys [repr]}]]
  (if repr
    (primitive-type repr)
    ::int))

(defn- coffitype->array-fn [type]
  (get
   {:coffi.mem/byte   `byte-array
    :coffi.mem/short  `short-array
    :coffi.mem/int    `int-array
    :coffi.mem/long   `long-array
    :coffi.mem/char   `char-array
    :coffi.mem/float  `float-array
    :coffi.mem/double `double-array}
   type
   `object-array))

(defn- coffitype->array-read-fn [type]
  ({:coffi.mem/byte   `read-bytes
    :coffi.mem/short  `read-shorts
    :coffi.mem/int    `read-ints
    :coffi.mem/long   `read-longs
    :coffi.mem/char   `read-chars
    :coffi.mem/float  `read-floats
    :coffi.mem/double `read-doubles} type))

(defmulti  generate-deserialize (fn [& xs] (if (vector? (first xs)) (ffirst xs) (first xs))))

(defmethod generate-deserialize :coffi.mem/byte     [_type offset segment-source-form] `(read-byte    ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/short    [_type offset segment-source-form] `(read-short   ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/int      [_type offset segment-source-form] `(read-int     ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/long     [_type offset segment-source-form] `(read-long    ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/char     [_type offset segment-source-form] `(read-char    ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/float    [_type offset segment-source-form] `(read-float   ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/double   [_type offset segment-source-form] `(read-double  ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/pointer  [_type offset segment-source-form] `(read-address ~segment-source-form ~offset))

(defmethod generate-deserialize :coffi.mem/c-string [_type offset segment-source-form]
  `(.getString (.reinterpret (.get ~(with-meta segment-source-form {:tag 'java.lang.foreign.MemorySegment}) pointer-layout ~offset) Integer/MAX_VALUE) 0))

(defn- generate-deserialize-array-as-array-bulk [array-type n offset segment-source-form]
  (list (coffitype->array-read-fn array-type) segment-source-form n offset))

(defn- generate-deserialize-array-as-array-inline [array-type n offset segment-source-form]
  (let [a (gensym 'array)]
    (concat
     `(let [~a (~(coffitype->array-fn array-type) ~n)])
     (map
      #(list `aset a % (generate-deserialize array-type (+ offset (* (size-of array-type) %)) segment-source-form))
      (range n))
     [a])))

(defn- generate-deserialize-array-as-array-loop [array-type n offset segment-source-form]
  (let [a (gensym 'array)]
    (concat
     `(let [~a (~(coffitype->array-fn array-type) ~n)])
     [(list `dotimes ['m n]
       (list `aset a 'm (generate-deserialize array-type `(+ ~offset (* ~(size-of array-type) ~'m)) segment-source-form)))]
     [a])))

(defn- generate-deserialize-array-as-array [array-type n offset segment-source-form]
  (if (coffitype->array-read-fn array-type)
    (generate-deserialize-array-as-array-bulk array-type n offset segment-source-form) ;bulk-copy available
    (let [inline-cutoff 32] ;this magic value has been benchmarked, but it may need adjusting for specific architectures
      (if (< n inline-cutoff)
        (generate-deserialize-array-as-array-inline array-type n offset segment-source-form)
        (generate-deserialize-array-as-array-loop array-type n offset segment-source-form)))))

(defn- generate-deserialize-array-as-vector-loop [array-type n offset segment-source-form]
  (let [loop-deserialize (generate-deserialize array-type `(+ ~offset (* ~(size-of array-type) ~'i)) segment-source-form)]
    `(loop [~'i 0 ~'v (transient [])]
       (if (< ~'i ~n)
         (recur (unchecked-inc ~'i) (conj! ~'v ~loop-deserialize))
         (persistent! ~'v)))))

(defn- generate-deserialize-array-as-vector-inline [array-type n offset segment-source-form]
  (vec (map #(generate-deserialize array-type (+ offset (* (size-of array-type) %)) segment-source-form) (range n))))

(defn- generate-deserialize-array-as-vector [array-type n offset segment-source-form]
  (let [inline-cutoff 64] ;this magic value has been benchmarked, but it may need adjusting for specific architectures
    (if (< n inline-cutoff)
      (generate-deserialize-array-as-vector-inline array-type n offset segment-source-form)
      (generate-deserialize-array-as-vector-loop array-type n offset segment-source-form))))

(defmethod generate-deserialize :coffi.mem/array    [[_ array-type n & {:keys [raw?]}] offset segment-source-form]
  (if raw?
    (generate-deserialize-array-as-array array-type n offset segment-source-form)
    (generate-deserialize-array-as-vector array-type n offset segment-source-form)))
