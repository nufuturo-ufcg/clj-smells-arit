(ns taoensso.encore
  "Extended core library for Clojure/Script that emphasizes:
    * Cross-platform API
    * Flexibility
    * Performance
    * Backwards compatibility

  This lib's mostly for my own use and for advanced users that feel
  comfortable reading this source. Not providing much beginner-oriented
  documentation for this, sorry.

  Common naming conventions used across my libs:
    **foo** - Dynamic var
    foo!    - Fn with side-effects, or that should otherwise be used cautiously
    foo?    - Truthy val or fn that returns truthy val
    foo!?   - Fn that has side-effects (or requires caution) and that return
              a truthy val. Note: !?, not ?!
    foo$    - Fn that's notably expensive to compute (e.g. hits db)
    foo_    - Derefable val (e.g. atom, volatile, delay, etc.)
    foo__   - Derefable in a derefable (e.g. delay in an atom), etc.
    _       - Unnamed val
    _foo    - Named but unused val
    ?foo    - Optional val (emphasize that val may be nil)
    foo*    - A variation of `foo` (e.g. `foo*` macro vs `foo` fn)
    foo'    - ''
    -foo    - Public implementation detail or intermediate (e.g. uncoerced) val
    >foo    - Val \"to   foo\" (e.g. >sender, >host), or fn to  put/coerce/transform
    <foo    - Val \"from foo\" (e.g. <sender, <host), or fn to take/coerce/transform
    ->foo   - Fn to put/coerce/transform

  Type affixes may be used for clarity:
    <prefix>-<name>  - m-users,   v-users,   n-users,    etc. (also nusers when unambiguous)
    <name>-<postfix> - users-map, users-vec, user-count, etc.

  Regarding name heirarchy:
    When there's a significant num of syms with a meaningful hierarchy,
    prefer names with descending hierarchy to emphasize structure and
    related groups/functionality, e.g.:
      `user-add`, `user-remove`, `user-mod` vs
      `add-user`, `remove-user`, `mod-user`, etc.

  Commit message tags (in priority order):
    [wip]   - Work-in-progress (still under development)

    [mod]   - Modify     behaviour (=>          breaking), [mod!], [mod!!], etc. for attention
    [fix]   - Fix broken behaviour (=> usu. non-breaking)
    [new]   - Add new    behaviour (=>      non-breaking)

    [doc]   - Documentation changes besides those better labelled as [mod], [fix], or [new]
    [nop]   - Other non-breaking changes (to implementation details, non-code changes, etc.)

    [x] [y] - Single commit with multiple tags (in priority order), try avoid

  Example commit messages:
    v1.0.0 (2022-01-27) ; Tagged release
    [new] [#122] Add new feature x (@contributor)

  Version numbers:
    Ver tables:  X.Y.Z (without backticks)
       Min ver: vX.Y.Z+
     Elsewhere: vX.Y.Z"

  {:author "Peter Taoussanis (@ptaoussanis)"}

  (:refer-clojure :exclude
   [if-let if-some if-not when when-not when-some when-let cond defonce
    binding run! some? ident? float? boolean? uri? indexed? bytes?
    int? pos-int? neg-int? nat-int? inst?
    simple-ident?   qualified-ident?
    simple-symbol?  qualified-symbol?
    simple-keyword? qualified-keyword?
    format update-in merge merge-with subvec
    memoize abs ex-message ex-data ex-cause ex-info
    newline satisfies? uuid
    pr prn print println])

  #?(:clj
     (:require
      [clojure.core    :as core]
      [clojure.string  :as str]
      [clojure.set     :as set]
      [clojure.java.io :as jio]
      [clojure.tools.reader.edn :as edn]
      [taoensso.truss :as truss :refer [ex-info]])

     :cljs
     (:require
      [cljs.core           :as core]
      [clojure.string      :as str]
      [clojure.set         :as set]
      [cljs.reader]
      [cljs.tools.reader.edn :as edn]
      [goog.object         :as gobj]
      [goog.array          :as garray]
      [goog.string         :as gstr]
      [goog.string.format]
      [goog.string.StringBuffer]
      [goog.events         :as gevents]
      [goog.net.XhrIo      :as gxhr]
      [goog.net.XhrIoPool  :as gxhr-pool]
      [goog.Uri.QueryData  :as gquery-data]
      [goog.net.EventType]
      [goog.net.ErrorCode]
      [taoensso.truss :as truss :refer [ex-info]]))

  #?(:clj
     (:import
      [java.util Date Locale TimeZone]
      [java.text SimpleDateFormat]
      [java.util.concurrent CountDownLatch]
      [java.util.concurrent.atomic AtomicReference]
      [java.util.function UnaryOperator])

     :cljs
     (:require-macros
      [taoensso.encore :as enc-macros :refer
       [compile-if qb
        if-let if-some if-not when when-not when-some when-let
        -cond cond cond! cond-let and? or? def* defonce
        -cas!? now-udt* now-nano* min* max*
        name-with-attrs deprecated new-object defalias
        identical-kw? satisfies? satisfies! instance! use-transient?
        with-default-print-opts typed-val]])))

(defn name-with-attrs
  "Given a symbol and args, returns [<name-with-attrs-meta> <args> <attrs>]
  with support for `defn` style `?docstring` and `?attrs-map`."
  ([sym args            ] (name-with-attrs sym args nil))
  ([sym args attrs-merge]
   (let [[?docstring args] (if (and (string? (first args)) (next args)) [(first args) (next args)] [nil        args])
         [attrs      args] (if (and (map?    (first args)) (next args)) [(first args) (next args)] [{}         args])
         [?docstring args] (if (and (string? (first args)) (next args)) [(first args) (next args)] [?docstring args])

         attrs (if ?docstring (assoc attrs :doc ?docstring) attrs)
         attrs (if-let [m (meta sym)] (conj m attrs) attrs)
         attrs (conj attrs attrs-merge)]

     [(with-meta sym attrs) args attrs])))

(defn finite-num?
  "Returns true iff given a number (of standard type) that is:
  finite (excl. NaN and infinities)."
  #?(:cljs {:tag 'boolean})
  [x]
  #?(:clj  (and (number? x) (Double/isFinite x)) ; Works with other types, incl. ratio
     :cljs (js/Number.isFinite x)
     #_
     (and
       (not ^boolean (js/isNaN x))
       #_(not (identical? x js/Infinity))
       (not (identical? x js/Number.POSITIVE_INFINITY))
       (not (identical? x js/Number.NEGATIVE_INFINITY)))))

(defn int?
  "Returns true iff given a number (of standard type) that is:
  a fixed-precision integer."
  #?(:cljs {:tag 'boolean})
  [x]
  #?(:clj
     (or
       (instance? Long    x)
       (instance? Integer x)
       (instance? Short   x)
       (instance? Byte    x))

     :cljs
     (and
       #_(number?   x)
       (finite-num? x)
       (== (js/parseFloat x) (js/parseInt x 10)))))

(defn float?
  "Returns true iff given a number (of standard type) that is:
  a fixed-precision floating-point (incl. NaN and infinities)."
  #?(:cljs {:tag 'boolean})
  [x]
  #?(:clj (or (instance? Double x) (instance? Float x))
     :cljs
     (and
       (number?        x)
       #_(finite?-num? x)
       (not (== (js/parseFloat x) (js/parseInt x 10))))))

(defn ensure-set [x] (if (set?    x) x (set x)))

(defn vsplit-last    [v] (let [c (count v)] (when (> c 0) [(when (> c 1) (pop v)) (peek v)])))

(defn- fsplit-last
  "Faster (f (vec (butlast xs)) (last x))."
  [xs f]
  (if (vector? xs)
    (let [[vn vl] (vsplit-last xs)] (f vn vl))
    (loop [butlast [] xs xs]
      (let [[x1 & xn] xs]
        (if xn
          (recur (conj butlast x1) xn)
          (f butlast x1))))))

(defn ks<=     #?(:cljs {:tag 'boolean}) [ks m]
  (let [counted-ks (if (counted? ks) ks (set ks))]
    (and
      (<= (count m)     (count counted-ks))
      (let [ks-set (ensure-set counted-ks)]
        (reduce-kv (fn [_ k v] (if (contains? ks-set k) true (reduced false))) true m)))))

(defn update-in
  "Like `core/update-in` but:.
    - Empty ks will return (f m), not act like [nil] ks.
    - Adds support for `not-found`.
    - Adds support for special return vals: `:update/dissoc`, `:update/abort`."
  ([m ks           f] (update-in m ks nil f))
  ([m ks not-found f]
   (if (empty? ks)
     (f m) ; m would also be a sensible choice, but (f m) is more useful
     (let [old (get-in m ks not-found)
           new (f old)]
       (case new
         (:update/abort  :swap/abort) m
         (:update/dissoc :swap/dissoc)
         (fsplit-last ks
           (fn [ks lk]
             (update-in m ks nil
               (fn [v]
                 (if v ; Assume associative
                   (dissoc v lk)
                   :update/abort)))))

         (assoc-in m ks new))))))

(defn- p! [m] (if (transient? m) (persistent! m) m))

(defn merge
  "Like `core/merge` but:
    - Supports `:merge/dissoc` vals.
    - Often faster, with much better worst-case performance."
  ([               ] nil)
  ([m1             ] (p! m1))
  ([m1 m2          ] (p!                                                     (merge-with* false nil m1 m2)))
  ([m1 m2 m3       ] (p!                              (merge-with* false nil (merge-with* false nil m1 m2) m3)))
  ([m1 m2 m3 & more] (p! (merge-with* false nil (cons (merge-with* false nil (merge-with* false nil m1 m2) m3) more)))))

(defn merge-with
  "Like `core/merge-with` but:
    - Supports `:merge/dissoc` vals.
    - Often faster, with much better worst-case performance."
  ([f                ] nil)
  ([f m1             ] (p! m1))
  ([f m1 m2          ] (p!                                                 (merge-with* false f m1 m2)))
  ([f m1 m2 m3       ] (p!                            (merge-with* false f (merge-with* false f m1 m2) m3)))
  ([f m1 m2 m3 & more] (p! (merge-with* false f (cons (merge-with* false f (merge-with* false f m1 m2) m3) more)))))

(defn clamp-int   ^long   [nmin nmax n] (let [nmin (long   nmin), nmax (long   nmax), n (long   n)] (if (< n nmin) nmin (if (> n nmax) nmax n))))

(defn abs [n]     (if (neg? n) (- n) n))

(defn inst?
  "Returns true iff given platform instant (`java.time.Instant` or `js/Date`)."
  #?(:cljs {:tag 'boolean})
  [x]
  #?(:clj  (instance? java.time.Instant x)
     :cljs (instance? js/Date           x)))

(defn str-builder
  "Returns a new stateful string builder:
    - `java.lang.StringBuilder`  for Clj
    - `goog.string.StringBuffer` for Cljs

  See also `sb-append`."
  #?(:clj  (^StringBuilder [    ]            (StringBuilder.)))
  #?(:cljs (               [    ] (goog.string.StringBuffer.)))
  #?(:clj  (^StringBuilder [init] (if (instance?            StringBuilder init) init            (StringBuilder. (str init)))))
  #?(:cljs (               [init] (if (instance? goog.string.StringBuffer init) init (goog.string.StringBuffer. (str init))))))

(defn sb-append
  "Appends given string/s to given string builder. See also `str-builder.`"
  (#?(:clj ^StringBuilder [^StringBuilder str-builder x]
      :cljs               [               str-builder x])
    (if (nil? x)
      (do      str-builder)
      (.append str-builder
        #?(:clj  (.toString ^Object x)
           :cljs (.toString         x)))))

  (#?(:clj  ^StringBuilder [^StringBuilder str-builder x & more]
      :cljs                [               str-builder x & more])
    (reduce
      (fn [acc in] (sb-append acc in))
      (sb-append str-builder x) more)))

(defn str-rf
  "String builder reducing fn."
  ([      ] (str-builder))
  ([acc   ] (str acc)) ; cf
  ([acc in]
   (if (nil? in)
     acc
     (.append (str-builder acc)
       #?(:clj  (.toString ^Object in)
          :cljs (.toString         in))))))

(defn str-starts-with?
  #?(:cljs {:tag 'boolean})
  [s substr]
  #?(:clj  (.startsWith ^String s ^String substr)
     :cljs (zero? (.indexOf s substr))))

(defn str-ends-with?
  #?(:cljs {:tag 'boolean})
  [s substr]
  #?(:clj (.endsWith ^String s ^String substr)
     :cljs
     (let [s-len      (.-length s)
           substr-len (.-length substr)]
       (when (>= s-len substr-len)
         (not= -1 (.indexOf s substr (- s-len substr-len)))))))

(defn str-join-once
  "Like `string/join` but skips nils and duplicate separators."
  {:tag #?(:clj 'String :cljs 'string)}
  [separator coll]
  (let [sep separator]
    (if (str/blank? sep)
      (str (reduce str-rf "" coll))
      (let [acc-ends-with-sep?_ (volatile! false)
            acc-empty?_         (volatile! true)]
        (str
          (reduce
            (fn [acc in]
              (if (nil? in)
                acc
                (let [in (str in)]
                  (if (= in "")
                    acc
                    (let [in-starts-with-sep? (str-starts-with? in sep)
                          in-ends-with-sep?   (str-ends-with?   in sep)
                          acc-ends-with-sep?  @acc-ends-with-sep?_
                          acc-empty?          @acc-empty?_]

                      (vreset! acc-ends-with-sep?_ in-ends-with-sep?)
                      (when acc-empty? (vreset! acc-empty?_ false))

                      (if acc-ends-with-sep?
                        (if in-starts-with-sep?
                          (sb-append acc (.substring in (count sep)))
                          (sb-append acc in))

                        (if in-starts-with-sep?
                          (sb-append acc in)
                          (if acc-empty?
                            (sb-append acc in)
                            (do (sb-append acc sep)
                                (sb-append acc in))))))))))
            (str-builder)
            coll))))))

(defn path
  {:tag #?(:clj 'String :cljs 'string)}
  [& parts] (str-join-once "/" parts))

(defn uuid
  "For Clj: returns a random `java.util.UUID`.
  For Cljs: returns a random UUID string.

  Uses strong randomness when possible.
  See also `uuid-str`, `nanoid`, `rand-id-fn`."
  {:inline #?(:default nil :clj (fn [] `(java.util.UUID/randomUUID)))}
  #?(:clj          (^java.util.UUID []  (java.util.UUID/randomUUID))
     :cljs
     ([]
      (if-let [f (oget js-?crypto "randomUUID")]
        (.call f js-?crypto)
        (let [^string quad-hex
              (fn []
                (let [unpadded-hex ^string (.toString (rand-int 65536) 16)]
                  (case (count   unpadded-hex)
                    1 (str "000" unpadded-hex)
                    2 (str "00"  unpadded-hex)
                    3 (str "0"   unpadded-hex)
                      (do        unpadded-hex))))

              ver-trip-hex ^string (.toString (bit-or 0x4000 (bit-and 0x0fff (rand-int 65536))) 16)
              res-trip-hex ^string (.toString (bit-or 0x8000 (bit-and 0x3fff (rand-int 65536))) 16)]

          (str (quad-hex) (quad-hex) "-" (quad-hex) "-" ver-trip-hex "-" res-trip-hex "-"
               (quad-hex) (quad-hex) (quad-hex)))))))

(defn uuid-str
  "Returns a random UUID string of given length (max 36).
  Uses strong randomness when possible. See also `uuid`, `nanoid`, `rand-id-fn`."
  {:tag #?(:clj 'String :cljs 'string)}
  ;; 128 bits of entropy with default length (36)
  ([max-len] (or (substr (uuid-str) :by-len 0 max-len) ""))
  ([       ]
   #?(:clj (str (uuid))
      :cljs     (uuid))))

(defn rand-bytes
  "Returns a random byte array of given size."
  #?(:clj ^bytes [secure? size] :cljs [prefer-secure? size])
  #?(:clj
     (let [ba (byte-array size)]
       (if secure?
         (.nextBytes (secure-rng) ba)
         (.nextBytes (java.util.concurrent.ThreadLocalRandom/current) ba))
       ba)

     :cljs
     (let [ba (js/Uint8Array. size)]
       (if-let [crypto (and prefer-secure? js-?crypto)]
         (.getRandomValues crypto ba)
         (dotimes [i size] (aset ba i (Math/floor (* 256 (js/Math.random))))))
       ba)))

(defn rand-id-fn
  "Returns a (fn rand-id []) that returns random id strings.
  Options include:
    `:chars`         - ∈ #{<string> :nanoid :alphanumeric :no-look-alikes ...}
    `:len`           - Length of id strings to generate
    `:rand-bytes-fn` - Optional (fn [size]) to return random byte array of given size

  See also `uuid-str`, `nano-id`."
  [{:keys [chars ^long len rand-bytes-fn]
    :or
    {chars         :nanoid
     len           21
     rand-bytes-fn (partial rand-bytes true)}}]

  (let [chars
        (case chars
          :alphanumeric    "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"   ; 62 chars
          :nanoid          "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_" ; 64 chars
          :nanoid-readable "346789ABCDEFGHJKLMNPQRTUVWXYabcdefghijkmnpqrtwxyz"                ; 49 chars, no look-alikes
          :numbers         "0123456789"                                                       ; 10 chars
          :alpha-lowercase "abcdefghijklmnopqrstuvwxyz"                                       ; 26 chars
          :alpha-uppercase "ABCDEFGHIJKLMNOPQRSTUVWXYZ"                                       ; 26 chars
          :hex-lowercase   "0123456789abcdef"                                                 ; 16 chars
          :hex-uppercase   "0123456789ABCDEF"                                                 ; 16 chars
          (truss/have string? chars))

        nchars       (count chars)
        max-char-idx (dec  nchars)
        chars
        #?(:clj  (.toCharArray (str chars))
           :cljs (object-array      chars))

        ;; (bit-and <rand-byte> <mask>) uniformly maps <rand-byte> (256 vals) to
        ;; the valid subset of <rand-char-idx> (<256) *without* bias. Downside is that
        ;; invalid char-idxs need to be discarded, hence the stepping mechanism and
        ;; need for extra bytes when nchars∤256.
        mask ; Set all bits except the most significant, Ref. <https://bit.ly/3dtYv73>
        (bit-and -1
          (dec
            (bit-shift-left 2
              (int (Math/floor (/ (Math/log (dec nchars)) (Math/log 2)))))))

        ;; Calculate size of random byte arrays to fill
        exp-bytes (double (/ (* mask len) nchars))
        stepn     (int (max 2 (Math/ceil (* 0.2 exp-bytes)))) ; Used iff step1 insufficient
        step1 ; Size of initial random byte array
        (int
          (if (zero? (int (mod 256 nchars)))
            len ; n|256 => no discarding (steps) needed
            (Math/ceil (* 1.2 exp-bytes))))]

    (fn rand-id []
      (let [sb (str-builder)]
        (loop [idx 0, max-idx (dec step1), ^bytes rand-bytes (rand-bytes-fn step1)]

          (let [possible-ch-idx (bit-and (aget rand-bytes idx) mask)]
            (when (<= possible-ch-idx max-char-idx)
              (.append sb (aget chars possible-ch-idx))))

          (cond
            (== (.length sb) len) (str sb)
            (== idx      max-idx) (recur 0 (dec stepn) (rand-bytes-fn stepn))
            :else (recur (unchecked-inc idx) max-idx rand-bytes)))))))

(defn ^:no-doc ^LightAtom latom
  "Private, don't use. Micro-optimized lightweight `atom`.
  Up to 30% faster than standard atoms, with the same atomicity guarantees."
  [init-state]
  (LightAtom.
    #?(:clj (AtomicReference. init-state)
       :cljs                  init-state)))

(defn memoize-last
  "Like `core/memoize` but only caches the given fn's latest input.
  Speeds repeated fn calls with the same arguments.
  Great for ReactJS render fn caching, etc."
  [f]
  (let [sentinel (new-object)
        call
        #?(:cljs
           (let [in_  (volatile! (new-object))
                 out_ (volatile! nil)]
             (fn [in* f0]
               (if (= in* @in_)
                 @out_
                 (let [out (f0)]
                   (vreset! in_  in*)
                   (vreset! out_ out)
                   out))))

           :clj
           (let [cache_ (java.util.concurrent.atomic.AtomicReference. nil)] ; [in out_]
             (fn [in* f0]
               (deref-cached!
                 [dv
                  (loop []
                    (let [current (.get cache_)]
                      (or
                        (when-let [[in out_] current] (when (= in in*) out_))
                        (let [dv (delay (f0))] (if (.compareAndSet cache_ current [in* dv]) dv (recur))))))]
                 (let [current (.get cache_)]
                   (when (identical? (second current) dv)
                     (.compareAndSet cache_ current nil)))))))]

    (fn memoized-fn
      ([            ] (call sentinel              (fn [] (f))))
      ([x           ] (call x                     (fn [] (f x))))
      ([x y         ] (call [sentinel x y       ] (fn [] (f x y))))
      ([x y z       ] (call [sentinel x y z     ] (fn [] (f x y z))))
      ([x y z & more] (call [sentinel x y z more] (fn [] (apply f x y z more)))))))

(defn fmemoize
  "For Clj: fastest possible memoize. Non-racey, 0-7 arity only.
  For Cljs: same as `core/memoize`."
  [f]
  #?(:cljs (cljs.core/memoize f)
     :clj
     ;; Non-racey just as fast as racey, and protects against nils in maps
     (let [sentinel (new-object)
           cache0_  (java.util.concurrent.atomic.AtomicReference. nil)
           cache1_  (java.util.concurrent.ConcurrentHashMap.)
           cachen_  (java.util.concurrent.ConcurrentHashMap.)]

       (fn
         ([ ] (deref-cached! [dv (or (.get cache0_) (let [dv (delay (f))] (if (.compareAndSet cache0_ nil dv) dv (.get cache0_))))] (.compareAndSet cache0_ dv nil)))
         ([x]
          (let [x* (if (identical? x nil) sentinel x)]
            (deref-cached!
              [dv
               (or
                 (.get cache1_ x*)
                 (let [dv (delay (f x))]
                   (or (.putIfAbsent cache1_ x* dv) dv)))]
              (.remove cache1_ x* dv))))

         ([x1 x2               ] (let [xs [x1 x2]               ] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2))               ] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))
         ([x1 x2 x3            ] (let [xs [x1 x2 x3]            ] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2 x3))            ] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))
         ([x1 x2 x3 x4         ] (let [xs [x1 x2 x3 x4]         ] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2 x3 x4))         ] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))
         ([x1 x2 x3 x4 x5      ] (let [xs [x1 x2 x3 x4 x5]      ] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2 x3 x4 x5))      ] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))
         ([x1 x2 x3 x4 x5 x6   ] (let [xs [x1 x2 x3 x4 x5 x6]   ] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2 x3 x4 x5 x6))   ] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))
         ([x1 x2 x3 x4 x5 x6 x7] (let [xs [x1 x2 x3 x4 x5 x6 x7]] (deref-cached! [dv (or (.get cachen_ xs) (let [dv (delay (f x1 x2 x3 x4 x5 x6 x7))] (or (.putIfAbsent cachen_ xs dv) dv)))] (.remove cachen_ xs dv))))))))

(defn- gc-now? [rate]
  #?(:clj  (<= (.nextDouble (java.util.concurrent.ThreadLocalRandom/current)) ^double rate)
     :cljs (<=       (.random js/Math)         rate)))

(deftype SimpleCacheEntry [delay ^long udt])

(deftype TickedCacheEntry
  [delay ^long udt
   ;; Mutable instead of updating cache map so that hits needn't swap.
   ;; Increments are racy (may drop) but ticks are only used as GC heuristics.
   #?(:clj ^:volatile-mutable ^long tick-lru, :cljs ^:mutable tick-lru)
   #?(:clj ^:volatile-mutable ^long tick-lfu, :cljs ^:mutable tick-lfu)]

  #?@(:clj
      [ITickedCacheEntry
       (tceTouch   [_ tick] (set! tick-lru tick) (set! tick-lfu (inc tick-lfu)))
       (tceTickSum [_]      (+    tick-lru                           tick-lfu))]))

(defn cache
  "Returns a cached version of given referentially transparent function `f`.

  Like `core/memoize` but:
    - Often faster, depending on options.
    - Prevents race conditions on writes.
    - Supports cache invalidation by prepending args with:
      - `:cache/del`   ; Delete cached item for subsequent args, returns nil.
      - `:cache/fresh` ; Renew  cached item for subsequent args, returns new val.

    - Supports options:
      - `ttl-ms` ; Expire cached items after <this> many msecs.
      - `size`   ; Restrict cache size to <this> many items at the next garbage
                 ; collection (GC).

      - `gc-every` ; Run garbage collection (GC) approximately once every
                   ; <this> many calls to cached fn. If unspecified, GC rate
                   ; will be determined automatically based on `size`.

  See also `defn-cached`, `fmemoize`, `memoize-last`."

  ([f] ; De-raced, commands
   #?(:cljs
      (let [cache_ (volatile! {})
            get-sentinel (js-obj)]

        (fn cached [& xs]
          (let [x1 (first xs)]
            (case x1

              (:cache/del :mem/del)
              (let [xn (next  xs)
                    x2 (first xn)]
                (if (case x2 (:cache/all :mem/all) true false)
                  (vreset! cache_ {})
                  (vswap!  cache_ dissoc xn))
                nil)

              (:cache/fresh :mem/fresh)
              (let [xn (next xs)
                    v  (apply f xn)]
                (vswap! cache_ assoc xn v) v)

              (let [v (get @cache_ xs get-sentinel)]
                (if (identical? v get-sentinel)
                  (let [v (apply f xs)] (vswap! cache_ assoc xs v) v)
                  v))))))

      :clj
      (let [nil-sentinel (Object.)
            cache_ (java.util.concurrent.ConcurrentHashMap.)]

        (fn
          ([ ] (deref-cached!
                 [dv
                  (or (.get cache_ nil-sentinel)
                    (let [dv (delay (f))]
                      (or (.putIfAbsent cache_ nil-sentinel dv) dv)))]
                 (.remove cache_ nil-sentinel dv)))

          ([& xs]
           (let [x1 (first xs)]
             (case x1

               (:cache/del :mem/del)
               (let [xn (next  xs)
                     x2 (first xn)]
                 (if (case x2 (:cache/all :mem/all) true false)
                   (.clear  cache_)
                   (.remove cache_ (or xn nil-sentinel)))
                 nil)

               (:cache/fresh :mem/fresh)
               (let [xn (next xs)
                     k  (or xn nil-sentinel)
                     dv (delay (apply f xn))]
                 (.put cache_ k dv)
                 (deref-cached! [dv dv] (.remove cache_ k dv)))

               (deref-cached!
                 [dv
                  (or (.get cache_ xs)
                    (let [dv (delay (apply f xs))]
                      (or (.putIfAbsent cache_ xs dv) dv)))]
                 (.remove cache_ xs dv)))))))))

  ([{:keys [size ttl-ms gc-every] :as opts} f]

   (truss/have? [:ks<= #{:size :ttl-ms :gc-every}] opts)
   (truss/have? [:or nil? pos-num?] size ttl-ms gc-every)

   (cond
     size ; De-raced, commands, ttl, gc, max-size
     (let [ticker   (counter)
           cache_   (latom nil) ; {<args> <TickedCacheEntry>}
           latch_   (latom nil) ; Used to pause writes during gc
           ttl-ms   (if ttl-ms (max 1 (long ttl-ms)) 0)
           ttl?     (not (zero? ttl-ms))
           size     (max 1 (long size))
           gc-every (max 1 (long (or gc-every (clamp-int 1000 16000 size))))]

       (fn cached [& args]
         (let [a1 (first args)]
           (case a1
             (:cache/del :mem/del)
             (let [argn (next args)
                   a2   (first argn)]
               (if (case a2 (:cache/all :mem/all) true false)
                 (reset! cache_ nil)
                 (cache_ #(dissoc % argn)))
               nil)

             (let [^long tick (ticker) ; Always inc, even on reads
                   instant (if ttl? (now-udt) 0)]

               (when (and
                       ;; We anyway have a tick, so may as well be exact
                       ;; (gc-now? gc-rate)
                       (== (rem tick gc-every) 0)
                       (>= (count (cache_)) (* 1.1 size)))

                 (let [latch #?(:clj (CountDownLatch. 1) :default nil)
                       udt-floor (- instant ttl-ms)]

                   (when (-cas!? latch_ nil latch)
                     (try
                       ;; First prune ttl-expired stuff
                       (when ttl?
                         (cache_
                           (fn swap-fn [m]
                             (persistent!
                               (reduce-kv
                                 (fn [acc k ^TickedCacheEntry e]
                                   (if (< (.-udt e) udt-floor)
                                     (dissoc! acc k)
                                     acc))
                                 (transient (or m {}))
                                 m)))))

                       ;; Then prune by ascending (worst) tick-sum:
                       (let [snapshot (cache_)
                             n-to-gc  (- (count snapshot) size)]

                         (when (>= n-to-gc (* 0.1 size))
                           (let [ks-to-gc
                                 (top n-to-gc
                                   (fn [k] (tce-tick-sum (get snapshot k)))
                                   (keys snapshot))]

                             (cache_
                               (fn swap-fn [m]
                                 (persistent!
                                   (reduce (fn [acc in] (dissoc! acc in))
                                     (transient (or m {})) ks-to-gc)))))))

                       nil
                       (finally
                         #?(:clj
                            (do
                              (reset!     latch_ nil)
                              (.countDown latch))
                            :default nil))))))

               (let [fresh? (case a1 (:cache/fresh :mem/fresh) true false)
                     args   (if fresh? (next args) args)

                     _ #?(:clj (when-let [l (latch_)] (.await ^CountDownLatch l)) :default nil)
                     ^TickedCacheEntry e0 (when-not fresh? (get (cache_) args))
                     ^TickedCacheEntry e
                     (if (and e0 (<= (- instant (.-udt e0)) ttl-ms))
                       e0 ; Common hit path, no swap needed
                       (cache_ args
                         (fn swap-fn [?e]
                           (if (or (nil? ?e) fresh?
                                   (> (- instant (.-udt ^TickedCacheEntry ?e)) ttl-ms))
                             (TickedCacheEntry. (delay (apply f args)) instant tick 0)
                             ?e))))]

                 (tce-touch! e tick)
                 #?(:clj
                    (deref-cached! [dv (.-delay e)]
                      (cache_
                        (fn [m]
                          (let [^TickedCacheEntry current (get m args)]
                            (if (and current (identical? (.-delay current) dv))
                              (dissoc m args)
                              m)))))
                    :cljs
                    @(.-delay e))))))))

     ttl-ms ; De-raced, commands, ttl, gc
     (let [gc-now? gc-now?
           cache_  (latom nil) ; {<args> <SimpleCacheEntry>}
           latch_  (latom nil) ; Used to pause writes during gc
           ttl-ms  (max 1 (long ttl-ms))
           gc-rate
           (let [gce (or gc-every 8e3)]
             (/ 1.0 (max 1 (long gce))))]

       (fn cached [& args]
         (let [a1 (first args)]
           (case a1
             (:cache/del :mem/del)
             (let [argn (next  args)
                   a2   (first argn)]
               (if (case a2 (:cache/all :mem/all) true false)
                 (reset! cache_ nil)
                 (cache_ #(dissoc % argn)))
               nil)

             (let [instant (now-udt)]

               (when (gc-now? gc-rate)
                 (let [latch #?(:clj (CountDownLatch. 1) :default nil)]
                   (when (-cas!? latch_ nil latch)
                     (try
                       (cache_
                         (fn swap-fn [m]
                           (persistent!
                             (reduce-kv
                               (fn [acc k ^SimpleCacheEntry e]
                                 (if (> (- instant (.-udt e)) ttl-ms)
                                   (dissoc! acc k)
                                   acc))
                               (transient (or m {}))
                               m))))

                       nil
                       (finally
                         #?(:clj
                            (do
                              (reset!     latch_ nil)
                              (.countDown latch))
                            :default nil))))))

               (let [fresh? (case a1 (:cache/fresh :mem/fresh) true false)
                     args   (if fresh? (next args) args)
                     _      #?(:clj (when-let [l (latch_)] (.await ^CountDownLatch l)) :default nil)
                     ^SimpleCacheEntry e0 (when-not fresh? (get (cache_) args))
                     ^SimpleCacheEntry e
                     (if (and e0 (<= (- instant (.-udt e0)) ttl-ms))
                       e0 ; Common hit path, no atomic update needed
                       (cache_ args
                         (fn swap-fn [?e]
                           (if (or (nil? ?e) fresh?
                                   (> (- instant (.-udt ^SimpleCacheEntry ?e)) ttl-ms))
                             (SimpleCacheEntry. (delay (apply f args)) instant)
                             ?e))))]
                 #?(:clj
                    (deref-cached! [dv (.-delay e)]
                      (cache_
                        (fn [m]
                          (let [^SimpleCacheEntry current (get m args)]
                            (if (and current (identical? (.-delay current) dv))
                              (dissoc m args)
                              m)))))
                    :cljs
                    @(.-delay e))))))))

     :else (cache f))))

(defn memoize
  "Alternative way to call `cache`, provided mostly for back compatibility.
  See `cache` docstring for details."
  ;; {:deprecated "Encore v3.36.0 (2022-11-18)"}
  ([            f] (cache                             f))
  ([     ttl-ms f] (cache {           :ttl-ms ttl-ms} f))
  ([size ttl-ms f] (cache {:size size :ttl-ms ttl-ms} f)))

(defn counter
  "Returns a fast atomic `Counter` with `init` initial integer value with:
    - @counter           => Return current val
    - (counter)          => Add 1 and return old val
    - (counter n)        => Add n and return old val
    - (counter action n) => Experimental, action ∈
        {:add :set :set-get :get-set :get-add :add-get}."
  ([    ] (counter 0))
  ([init]
   #?(:clj  (Counter. (java.util.concurrent.atomic.AtomicLong. (long init)))
      :cljs (Counter.                                          (long init)))))

(defn top-into
  "Conjoins the top `n` items from `coll` into `to` using `reduce-top`."
  ([to n           coll] (top-into to n identity compare coll))
  ([to n keyfn     coll] (top-into to n keyfn    compare coll))
  ([to n keyfn cmp coll]
   (if (use-transient? ^long n to)
     (persistent! (reduce-top n keyfn cmp conj! (transient to) coll))
     (do          (reduce-top n keyfn cmp conj             to  coll)))))

(defn top
  "Returns a sorted vector of the top `n` items from `coll` using `reduce-top`."
  ([n           coll] (top-into [] n identity compare coll))
  ([n keyfn     coll] (top-into [] n keyfn    compare coll))
  ([n keyfn cmp coll] (top-into [] n keyfn    cmp     coll)))
