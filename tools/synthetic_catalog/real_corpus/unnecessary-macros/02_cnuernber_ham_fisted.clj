(ns ham-fisted.api
  "Fast mutable and immutable associative data structures based on bitmap trie
  hashmaps. Mutable pathways implement the `java.util.Map` or `Set` interfaces
  including in-place update features such as compute or computeIfPresent.

  Mutable maps or sets can be turned into their immutable counterparts via the
  Clojure `persistent!` call. This allows working in a mutable space for
  convenience and performance then switching to an immutable pathway when
  necessary. Note: after `persistent!` one should never backdoor mutate map or
  set again as this will break the contract of immutability.  Immutable
  data structures also support conversion to transient via `transient`.

  Map keysets (`.keySet`) are full `PersistentHashSet`s of keys.

  Maps and sets support metadata but setting the metadata on mutable objects
  returns a new mutable object that shares the backing store leading to possible
  issues. Metadata is transferred to the persistent versions of the
  mutable/transient objects upon `persistent!`.

  Very fast versions of union, difference and intersection are provided for maps
  and sets with the map version of union and difference requiring an extra
  argument, a `java.util.BiFunction` or an `IFn` taking 2 arguments to merge the
  left and right sides into the final map. These implementations of union,
  difference, and intersection are the fastest implementation of these
  operations we know of on the JVM.

  Additionally a fast value update pathway is provided, enabling quickly
  updating all the values in a given map. Additionally, a new map primitive
  - [[mapmap]] - allows transforming a given map into a new map quickly by
  mapping across all the entries.

  Unlike the standard Java objects, mutation-via-iterator is not supported."
  (:require [ham-fisted.iterator :as iterator]
            [ham-fisted.function :refer [bi-function ->bi-function function bi-consumer obj->long]
             :as hamf-fn]
            [ham-fisted.lazy-noncaching
             :refer [map concat filter repeatedly]
             :as lznc]
            [ham-fisted.caffeine :as hamf-caffeine]
            [ham-fisted.lazy-caching :as lzc]
            [ham-fisted.alists :as alists]
            [ham-fisted.impl :as impl]
            [ham-fisted.reduce :refer [reduce-reducer preduce options->parallel-options preduce-reducer
                                       long-accumulator double-accumulator double-consumer-accumulator
                                       long-consumer-accumulator]
             :as hamf-rf]
            [ham-fisted.language :as hamf-language]
            [ham-fisted.protocols :as protocols]
            [ham-fisted.defprotocol :refer [extend extend-type extend-protocol]])
  (:import [ham_fisted UnsharedHashMap UnsharedLongHashMap UnsharedHashSet
            PersistentHashSet PersistentHashMap PersistentLongHashMap
            ArrayLists$ArrayOwner MergeIterator
            HashProvider MapSetOps SetOps ObjArray UpdateValues
            MutList ImmutList StringCollection ArrayImmutList ArrayLists
            ImmutSort IMutList Ranges$LongRange ArrayHelpers
            Ranges$DoubleRange IFnDef Transformables$MapIterable
            Transformables$FilterIterable Transformables$CatIterable
            Transformables$MapList Transformables$IMapable Transformables
            ReindexList ConstList ArrayLists$ObjectArrayList Transformables$SingleMapList
            ArrayLists$IntArrayList ArrayLists$LongArrayList ArrayLists$DoubleArrayList
            ReverseList TypedList DoubleMutList LongMutList
            Consumers Sum Sum$SimpleSum Casts Reducible IndexedDoubleConsumer
            IndexedLongConsumer IndexedConsumer ITypedReduce ParallelOptions Reductions
            IFnDef$LO IFnDef$LL IFnDef$DO IFnDef$DD IFnDef$DDD
            IFnDef$LLL ParallelOptions$CatParallelism IFnDef$OO IFnDef$OOO IFnDef$ODO
            IFnDef$OLO IFnDef$OD IFnDef$OL IFnDef$LD IFnDef$DL IFnDef$OLOO IFnDef$OLDO
            IFnDef$OLLO IFnDef$LongPredicate IFnDef$DoublePredicate IFnDef$Predicate
            Consumers$IncConsumer Reductions$IndexedDoubleAccum Reductions$IndexedLongAccum
            Reductions$IndexedAccum MutableMap IAMapEntry MapForward TypedNth
            TreeList MutTreeList]
           [ham_fisted.alists ByteArrayList ShortArrayList CharArrayList FloatArrayList
            BooleanArrayList]
           [clojure.lang ITransientAssociative2 ITransientCollection Indexed
            IEditableCollection RT IPersistentMap Associative Util IFn ArraySeq
            Reversible IReduce IReduceInit IFn$DD IFn$DL IFn$DO IFn$LD IFn$LL IFn$LO
            IFn$OD IFn$OL IFn$OLO IFn$ODO IObj Util IReduceInit Seqable IteratorSeq
            ITransientMap Counted Box]
           [java.util Map Map$Entry List RandomAccess Set Collection ArrayList Arrays
            Comparator Random Collections Iterator PriorityQueue LinkedHashMap LongSummaryStatistics
            DoubleSummaryStatistics]
           [java.lang.reflect Array]
           [java.util.function Function BiFunction BiConsumer Consumer
            DoubleBinaryOperator LongBinaryOperator LongFunction IntFunction
            DoubleConsumer DoublePredicate DoubleUnaryOperator LongPredicate
            LongUnaryOperator LongConsumer Predicate UnaryOperator]
           [java.util.concurrent ForkJoinPool ExecutorService Callable Future
            ConcurrentHashMap ForkJoinTask ArrayBlockingQueue]
           [it.unimi.dsi.fastutil.ints IntComparator IntArrays]
           [it.unimi.dsi.fastutil.longs LongComparator]
           [it.unimi.dsi.fastutil.floats FloatComparator]
           [it.unimi.dsi.fastutil.doubles DoubleComparator DoubleArrays]
           [it.unimi.dsi.fastutil.objects ObjectArrays]
           [com.github.benmanes.caffeine.cache Caffeine LoadingCache CacheLoader Cache
            RemovalCause]
           [com.github.benmanes.caffeine.cache.stats CacheStats]
           [java.time Duration]
           [java.util.logging Logger]
           [java.util.stream IntStream DoubleStream])
  (:refer-clojure :exclude [assoc! conj! frequencies merge merge-with memoize cond
                            into hash-map not
                            group-by subvec group-by mapv vec vector object-array
                            sort int-array long-array double-array float-array
                            range map concat filter filterv first last pmap take take-last drop
                            drop-last sort-by repeat repeatedly shuffle into-array
                            empty? reverse byte-array short-array char-array boolean-array
                            keys vals persistent! rest transient update-vals
                            re-matches complement count extend extend-type extend-protocol
                            constantly]))

(defmacro inth
  "nth operation returning a primitive int.  Efficient when obj is an int array."
  [obj idx]
  `(TypedNth/inth ~obj ~idx))
