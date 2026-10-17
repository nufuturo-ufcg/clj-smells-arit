(ns cljs.pprint
  (:refer-clojure :exclude [deftype print println pr prn float?])
  (:require-macros
    [cljs.pprint :as m :refer [with-pretty-writer getf setf deftype
                               pprint-logical-block print-length-loop
                               defdirectives formatter-out]])
  (:require
    [cljs.core :refer [IWriter IDeref]]
    [clojure.string :as string]
    [goog.string :as gstring])
  (:import [goog.string StringBuffer]))

(defn- print [& more]
  (-write *out* (apply print-str more)))

(defn- println [& more]
  (apply print more)
  (-write *out* \newline))

(defn- ^:dynamic pr [& more]
  (-write *out* (apply pr-str more)))

(defn- prn [& more]
  (apply pr more)
  (-write *out* \newline))

(defn- map-passing-context [func initial-context lis]
  (loop [context initial-context
         lis lis
         acc []]
    (if (empty? lis)
      [acc context]
      (let [this (first lis)
            remainder (next lis)
            [result new-context] (apply func [this context])]
        (recur new-context remainder (conj acc result))))))

(defn- consume [func initial-context]
  (loop [context initial-context
         acc []]
    (let [[result new-context] (apply func [context])]
      (if (not result)
        [acc new-context]
        (recur new-context (conj acc result))))))

(defn- unzip-map [m]
  "Take a  map that has pairs in the value slots and produce a pair of maps,
   the first having all the first elements of the pairs and the second all
   the second elements of the pairs"
  [(into {} (for [[k [v1 v2]] m] [k v1]))
   (into {} (for [[k [v1 v2]] m] [k v2]))])

(defn- tuple-map [m v1]
  "For all the values, v, in the map, replace them with [v v1]"
  (into {} (for [[k v] m] [k [v v1]])))

(defn- prefix-count [aseq val]
  "Return the number of times that val occurs at the start of sequence aseq,
if val is a seq itself, count the number of times any element of val occurs at the
beginning of aseq"
  (let [test (if (coll? val) (set val) #{val})]
    (loop [pos 0]
      (if (or (= pos (count aseq)) (not (test (nth aseq pos))))
        pos
        (recur (inc pos))))))

(defprotocol IPrettyFlush
  (-ppflush [pp]))

(def ^:dynamic ^{:private true} *default-page-width* 72)

(defn- get-field [this sym]
  (sym @@this))

(defn- set-field [this sym new-val]
  (swap! @this assoc sym new-val))

(defn- get-column [this]
  (get-field this :cur))

(defn- get-max-column [this]
  (get-field this :max))

(defn- c-write-char [this c]
  (if (= c \newline)
    (do
      (set-field this :cur 0)
      (set-field this :line (inc (get-field this :line))))
    (set-field this :cur (inc (get-field this :cur))))
  (-write (get-field this :base) c))

(defn- column-writer
  ([writer] (column-writer writer *default-page-width*))
  ([writer max-columns]
   (let [fields (atom {:max max-columns, :cur 0, :line 0 :base writer})]
     (reify

       IDeref
       (-deref [_] fields)

       IWriter
       (-flush [_]
         (-flush writer))
       (-write
         ;;-write isn't multi-arity, so need different way to do this
         #_([this ^chars cbuf ^Number off ^Number len]
          (let [writer (get-field this :base)]
            (-write writer cbuf off len)))
         [this x]
         (condp = (type x)
           js/String
           (let [s x
                 nl (.lastIndexOf s \newline)]
             (if (neg? nl)
               (set-field this :cur (+ (get-field this :cur) (count s)))
               (do
                 (set-field this :cur (- (count s) nl 1))
                 (set-field this :line (+ (get-field this :line)
                                          (count (filter #(= % \newline) s))))))
             (-write (get-field this :base) s))
           js/Number
           (c-write-char this x)))))))

(defrecord ^{:private true} logical-block
  [parent section start-col indent
   done-nl intra-block-nl
   prefix per-line-prefix suffix
   logical-block-callback])

(defn- ancestor? [parent child]
  (loop [child (:parent child)]
    (cond
      (nil? child) false
      (identical? parent child) true
      :else (recur (:parent child)))))

(defn- buffer-length [l]
  (let [l (seq l)]
    (if l
      (- (:end-pos (last l)) (:start-pos (first l)))
      0)))

(deftype buffer-blob :data :trailing-white-space :start-pos :end-pos)

(deftype nl-t :type :logical-block :start-pos :end-pos)

(deftype start-block-t :logical-block :start-pos :end-pos)

(deftype end-block-t :logical-block :start-pos :end-pos)

(deftype indent-t :logical-block :relative-to :offset :start-pos :end-pos)

(def ^:private pp-newline (fn [] "\n"))

(defmulti ^{:private true} write-token #(:type-tag %2))

(defmethod write-token :start-block-t [this token]
  (when-let [cb (getf :logical-block-callback)] (cb :start))
  (let [lb (:logical-block token)]
    (when-let [prefix (:prefix lb)]
      (-write (getf :base) prefix))
    (let [col (get-column (getf :base))]
      (reset! (:start-col lb) col)
      (reset! (:indent lb) col))))

(defmethod write-token :end-block-t [this token]
  (when-let [cb (getf :logical-block-callback)] (cb :end))
  (when-let [suffix (:suffix (:logical-block token))]
    (-write (getf :base) suffix)))

(defmethod write-token :indent-t [this token]
  (let [lb (:logical-block token)]
    (reset! (:indent lb)
            (+ (:offset token)
               (condp = (:relative-to token)
                 :block @(:start-col lb)
                 :current (get-column (getf :base)))))))

(defmethod write-token :buffer-blob [this token]
  (-write (getf :base) (:data token)))

(defmethod write-token :nl-t [this token]
  (if (or (= (:type token) :mandatory)
          (and (not (= (:type token) :fill))
               @(:done-nl (:logical-block token))))
    (emit-nl this token)
    (if-let [tws (getf :trailing-white-space)]
      (-write (getf :base) tws)))
  (setf :trailing-white-space nil))

(defn- write-tokens [this tokens force-trailing-whitespace]
  (doseq [token tokens]
    (if-not (= (:type-tag token) :nl-t)
      (if-let [tws (getf :trailing-white-space)]
        (-write (getf :base) tws)))
    (write-token this token)
    (setf :trailing-white-space (:trailing-white-space token))
    (let [tws (getf :trailing-white-space)]
      (when (and force-trailing-whitespace tws)
        (-write (getf :base) tws)
        (setf :trailing-white-space nil)))))

(defn- tokens-fit? [this tokens]
  (let [maxcol (get-max-column (getf :base))]
    (or
      (nil? maxcol)
      (< (+ (get-column (getf :base)) (buffer-length tokens)) maxcol))))

(defn- linear-nl? [this lb section]
  (or @(:done-nl lb)
      (not (tokens-fit? this section))))

(defn- miser-nl? [this lb section]
  (let [miser-width (get-miser-width this)
        maxcol (get-max-column (getf :base))]
    (and miser-width maxcol
         (>= @(:start-col lb) (- maxcol miser-width))
         (linear-nl? this lb section))))

(defmulti ^{:private true} emit-nl? (fn [t _ _ _] (:type t)))

(defmethod emit-nl? :linear [newl this section _]
  (let [lb (:logical-block newl)]
    (linear-nl? this lb section)))

(defmethod emit-nl? :miser [newl this section _]
  (let [lb (:logical-block newl)]
    (miser-nl? this lb section)))

(defmethod emit-nl? :fill [newl this section subsection]
  (let [lb (:logical-block newl)]
    (or @(:intra-block-nl lb)
        (not (tokens-fit? this subsection))
        (miser-nl? this lb section))))

(defmethod emit-nl? :mandatory [_ _ _ _]
  true)

(defn- get-section [buffer]
  (let [nl (first buffer)
        lb (:logical-block nl)
        section (seq (take-while #(not (and (nl-t? %) (ancestor? (:logical-block %) lb)))
                                 (next buffer)))]
    [section (seq (drop (inc (count section)) buffer))]))

(defn- get-sub-section [buffer]
  (let [nl (first buffer)
        lb (:logical-block nl)
        section (seq (take-while #(let [nl-lb (:logical-block %)]
                                   (not (and (nl-t? %) (or (= nl-lb lb) (ancestor? nl-lb lb)))))
                                 (next buffer)))]
    section))

(defn- update-nl-state [lb]
  (reset! (:intra-block-nl lb) true)
  (reset! (:done-nl lb) true)
  (loop [lb (:parent lb)]
    (if lb
      (do (reset! (:done-nl lb) true)
          (reset! (:intra-block-nl lb) true)
          (recur (:parent lb))))))

(defn- emit-nl [this nl]
  (-write (getf :base) (pp-newline))
  (setf :trailing-white-space nil)
  (let [lb (:logical-block nl)
        prefix (:per-line-prefix lb)]
    (if prefix
      (-write (getf :base) prefix))
    (let [istr (apply str (repeat (- @(:indent lb) (count prefix)) \space))]
      (-write (getf :base) istr))
    (update-nl-state lb)))

(defn- split-at-newline [tokens]
  (let [pre (seq (take-while #(not (nl-t? %)) tokens))]
    [pre (seq (drop (count pre) tokens))]))

(defn- write-token-string [this tokens]
  (let [[a b] (split-at-newline tokens)]
    (if a (write-tokens this a false))
    (if b
      (let [[section remainder] (get-section b)
            newl (first b)]
        (let [do-nl (emit-nl? newl this section (get-sub-section b))
              result (if do-nl
                       (do
                         (emit-nl this newl)
                         (next b))
                       b)
              long-section (not (tokens-fit? this result))
              result (if long-section
                       (let [rem2 (write-token-string this section)]
                         (if (= rem2 section)
                           (do ; If that didn't produce any output, it has no nls
                             ; so we'll force it
                             (write-tokens this section false)
                             remainder)
                           (into [] (concat rem2 remainder))))
                       result)]
          result)))))

(defn- write-line [this]
  (loop [buffer (getf :buffer)]
    (setf :buffer (into [] buffer))
    (if (not (tokens-fit? this buffer))
      (let [new-buffer (write-token-string this buffer)]
        (if-not (identical? buffer new-buffer)
          (recur new-buffer))))))

(defn- add-to-buffer [this token]
  (setf :buffer (conj (getf :buffer) token))
  (if (not (tokens-fit? this (getf :buffer)))
    (write-line this)))

(defn- write-buffered-output [this]
  (write-line this)
  (if-let [buf (getf :buffer)]
    (do
      (write-tokens this buf true)
      (setf :buffer []))))

(defn- write-white-space [this]
  (when-let [tws (getf :trailing-white-space)]
    (-write (getf :base) tws)
    (setf :trailing-white-space nil)))

(defn- write-initial-lines
  [^Writer this ^String s]
  (let [lines (string/split s "\n" -1)]
    (if (= (count lines) 1)
      s
      (let [^String prefix (:per-line-prefix (first (getf :logical-blocks)))
            ^String l (first lines)]
        (if (= :buffering (getf :mode))
          (let [oldpos (getf :pos)
                newpos (+ oldpos (count l))]
            (setf :pos newpos)
            (add-to-buffer this (make-buffer-blob l nil oldpos newpos))
            (write-buffered-output this))
          (do
            (write-white-space this)
            (-write (getf :base) l)))
        (-write (getf :base) \newline)
        (doseq [^String l (next (butlast lines))]
          (-write (getf :base) l)
          (-write (getf :base) (pp-newline))
          (if prefix
            (-write (getf :base) prefix)))
        (setf :buffering :writing)
        (last lines)))))

(defn- p-write-char [this c]
  (if (= (getf :mode) :writing)
    (do
      (write-white-space this)
      (-write (getf :base) c))
    (if (= c \newline)
      (write-initial-lines this \newline)
      (let [oldpos (getf :pos)
            newpos (inc oldpos)]
        (setf :pos newpos)
        (add-to-buffer this (make-buffer-blob (char c) nil oldpos newpos))))))

(defn- pretty-writer [writer max-columns miser-width]
  (let [lb (logical-block. nil nil (atom 0) (atom 0) (atom false) (atom false)
                           nil nil nil nil)
        ; NOTE: may want to just `specify!` #js { ... fields ... } with the protocols
        fields (atom {:pretty-writer true
                      :base (column-writer writer max-columns)
                      :logical-blocks lb
                      :sections nil
                      :mode :writing
                      :buffer []
                      :buffer-block lb
                      :buffer-level 1
                      :miser-width miser-width
                      :trailing-white-space nil
                      :pos 0})]
    (reify

      IDeref
      (-deref [_] fields)

      IWriter
      (-write [this x]
        (condp = (type x)
          js/String
          (let [s0 (write-initial-lines this x)
                s (string/replace-first s0 #"\s+$" "")
                white-space (subs s0 (count s))
                mode (getf :mode)]
            (if (= mode :writing)
              (do
                (write-white-space this)
                (-write (getf :base) s)
                (setf :trailing-white-space white-space))
              (let [oldpos (getf :pos)
                    newpos (+ oldpos (count s0))]
                (setf :pos newpos)
                (add-to-buffer this (make-buffer-blob s white-space oldpos newpos)))))
          js/Number
          (p-write-char this x)))
      (-flush [this]
        (-ppflush this)
        (-flush (getf :base)))

      IPrettyFlush
      (-ppflush [this]
        (if (= (getf :mode) :buffering)
          (do
            (write-tokens this (getf :buffer) true)
            (setf :buffer []))
          (write-white-space this)))

      )))

(defn- nl [this type]
  (setf :mode :buffering)
  (let [pos (getf :pos)]
    (add-to-buffer this (make-nl-t type (getf :logical-blocks) pos pos))))

(defn- indent [this relative-to offset]
  (let [lb (getf :logical-blocks)]
    (if (= (getf :mode) :writing)
      (do
        (write-white-space this)
        (reset! (:indent lb)
                (+ offset (condp = relative-to
                            :block @(:start-col lb)
                            :current (get-column (getf :base))))))
      (let [pos (getf :pos)]
        (add-to-buffer this (make-indent-t lb relative-to offset pos pos))))))

(defn- get-miser-width [this]
  (getf :miser-width))

(def ^:dynamic ^{:private true} *current-level* 0)

(def ^:dynamic ^{:private true} *current-length* nil)

(defn- pretty-writer?
  "Return true iff x is a PrettyWriter"
  [x] (and (satisfies? IDeref x) (:pretty-writer @@x)))

(defn write-out
  "Write an object to *out* subject to the current bindings of the printer control
variables. Use the kw-args argument to override individual variables for this call (and
any recursive calls).

*out* must be a PrettyWriter if pretty printing is enabled. This is the responsibility
of the caller.

This method is primarily intended for use by pretty print dispatch functions that
already know that the pretty printer will have set up their environment appropriately.
Normal library clients should use the standard \"write\" interface. "
  [object]
  (let [length-reached (and *current-length*
                            *print-length*
                            (>= *current-length* *print-length*))]
    (if-not *print-pretty*
      (pr object)
      (if length-reached
        (-write *out* "...") ;;TODO could this (incorrectly) print ... on the next line?
        (do
          (if *current-length* (set! *current-length* (inc *current-length*)))
          (*print-pprint-dispatch* object))))
    length-reached))

(defn write
  "Write an object subject to the current bindings of the printer control variables.
Use the kw-args argument to override individual variables for this call (and any
recursive calls). Returns the string result if :stream is nil or nil otherwise.

The following keyword arguments can be passed with values:
  Keyword              Meaning                              Default value
  :stream              Writer for output or nil             true (indicates *out*)
  :base                Base to use for writing rationals    Current value of *print-base*
  :circle*             If true, mark circular structures    Current value of *print-circle*
  :length              Maximum elements to show in sublists Current value of *print-length*
  :level               Maximum depth                        Current value of *print-level*
  :lines*              Maximum lines of output              Current value of *print-lines*
  :miser-width         Width to enter miser mode            Current value of *print-miser-width*
  :dispatch            The pretty print dispatch function   Current value of *print-pprint-dispatch*
  :pretty              If true, do pretty printing          Current value of *print-pretty*
  :radix               If true, prepend a radix specifier   Current value of *print-radix*
  :readably*           If true, print readably              Current value of *print-readably*
  :right-margin        The column for the right margin      Current value of *print-right-margin*
  :suppress-namespaces If true, no namespaces in symbols    Current value of *print-suppress-namespaces*

  * = not yet supported
"
  [object & kw-args]
  (let [options (merge {:stream true} (apply hash-map kw-args))]
    ;;TODO rewrite this as a macro
    (binding [cljs.pprint/*print-base* (:base options cljs.pprint/*print-base*)
              ;;:case             *print-case*,
              cljs.pprint/*print-circle* (:circle options cljs.pprint/*print-circle*)
              ;;:escape           *print-escape*
              ;;:gensym           *print-gensym*
              cljs.core/*print-length* (:length options cljs.core/*print-length*)
              cljs.core/*print-level* (:level options cljs.core/*print-level*)
              cljs.pprint/*print-lines* (:lines options cljs.pprint/*print-lines*)
              cljs.pprint/*print-miser-width* (:miser-width options cljs.pprint/*print-miser-width*)
              cljs.pprint/*print-pprint-dispatch* (:dispatch options cljs.pprint/*print-pprint-dispatch*)
              cljs.pprint/*print-pretty* (:pretty options cljs.pprint/*print-pretty*)
              cljs.pprint/*print-radix* (:radix options cljs.pprint/*print-radix*)
              cljs.core/*print-readably* (:readably options cljs.core/*print-readably*)
              cljs.pprint/*print-right-margin* (:right-margin options cljs.pprint/*print-right-margin*)
              cljs.pprint/*print-suppress-namespaces* (:suppress-namespaces options cljs.pprint/*print-suppress-namespaces*)]
      ;;TODO enable printing base
      #_[bindings (if (or (not (= *print-base* 10)) *print-radix*)
                  {#'pr pr-with-base}
                  {})]
      (binding []
        (let [sb (StringBuffer.)
              optval (if (contains? options :stream)
                       (:stream options)
                       true)
              base-writer (if (or (true? optval) (nil? optval))
                            (StringBufferWriter. sb)
                            optval)]
          (if *print-pretty*
            (with-pretty-writer base-writer
                                (write-out object))
            (binding [*out* base-writer]
              (pr object)))
          (if (true? optval)
            (string-print (str sb)))
          (if (nil? optval)
            (str sb)))))))

(defn pprint
  ([object]
   (let [sb (StringBuffer.)]
     (binding [*out* (StringBufferWriter. sb)]
       (pprint object *out*)
       (string-print (str sb)))))
  ([object writer]
   (with-pretty-writer writer
                       (binding [*print-pretty* true]
                         (write-out object))
                       (if (not (= 0 (get-column *out*)))
                         (-write *out* \newline)))))

(defn- check-enumerated-arg [arg choices]
  (if-not (choices arg)
    ;; TODO clean up choices string
    (throw (js/Error. (str "Bad argument: " arg ". It must be one of " choices)))))

(defn pprint-newline
  "Print a conditional newline to a pretty printing stream. kind specifies if the
  newline is :linear, :miser, :fill, or :mandatory.

  This function is intended for use when writing custom dispatch functions.

  Output is sent to *out* which must be a pretty printing writer."
  [kind]
  (check-enumerated-arg kind #{:linear :miser :fill :mandatory})
  (nl *out* kind))

(defn pprint-indent
  "Create an indent at this point in the pretty printing stream. This defines how
following lines are indented. relative-to can be either :block or :current depending
whether the indent should be computed relative to the start of the logical block or
the current column position. n is an offset.

This function is intended for use when writing custom dispatch functions.

Output is sent to *out* which must be a pretty printing writer."
  [relative-to n]
  (check-enumerated-arg relative-to #{:block :current})
  (indent *out* relative-to n))

(defn cl-format
  "An implementation of a Common Lisp compatible format function. cl-format formats its
arguments to an output stream or string based on the format control string given. It
supports sophisticated formatting of structured data.

Writer satisfies IWriter, true to output via *print-fn* or nil to output
to a string, format-in is the format control string and the remaining arguments
are the data to be formatted.

The format control string is a string to be output with embedded 'format directives'
describing how to format the various arguments passed in.

If writer is nil, cl-format returns the formatted result string. Otherwise, cl-format
returns nil.

For example:
 (let [results [46 38 22]]
        (cl-format true \"There ~[are~;is~:;are~]~:* ~d result~:p: ~{~d~^, ~}~%\"
                   (count results) results))

Prints via *print-fn*:
 There are 3 results: 46, 38, 22

Detailed documentation on format control strings is available in the \"Common Lisp the
Language, 2nd edition\", Chapter 22 (available online at:
http://www.cs.cmu.edu/afs/cs.cmu.edu/project/ai-repository/ai/html/cltl/clm/node200.html#SECTION002633000000000000000)
and in the Common Lisp HyperSpec at
http://www.lispworks.com/documentation/HyperSpec/Body/22_c.htm"
  {:see-also [["http://www.cs.cmu.edu/afs/cs.cmu.edu/project/ai-repository/ai/html/cltl/clm/node200.html#SECTION002633000000000000000"
               "Common Lisp the Language"]
              ["http://www.lispworks.com/documentation/HyperSpec/Body/22_c.htm"
               "Common Lisp HyperSpec"]]}
  [writer format-in & args]
  (let [compiled-format (if (string? format-in) (compile-format format-in) format-in)
        navigator (init-navigator args)]
    (execute-format writer compiled-format navigator)))

(def ^:dynamic ^{:private true} *format-str* nil)

(defn- format-error [message offset]
  (let [full-message (str message \newline *format-str* \newline
                          (apply str (repeat offset \space)) "^" \newline)]
    (throw (js/Error full-message))))

(defrecord ^{:private true}
  arg-navigator [seq rest pos])

(defn- init-navigator
  "Create a new arg-navigator from the sequence with the position set to 0"
  {:skip-wiki true}
  [s]
  (let [s (seq s)]
    (arg-navigator. s s 0)))

(defn- next-arg [navigator]
  (let [rst (:rest navigator)]
    (if rst
      [(first rst) (arg-navigator. (:seq navigator) (next rst) (inc (:pos navigator)))]
      (throw (js/Error "Not enough arguments for format definition")))))

(defrecord ^{:private true}
  compiled-directive [func def params offset])

(defn- realize-parameter [[param [raw-val offset]] navigator]
  (let [[real-param new-navigator]
        (cond
          (contains? #{:at :colon} param) ;pass flags through unchanged - this really isn't necessary
          [raw-val navigator]

          (= raw-val :parameter-from-args)
          (next-arg navigator)

          (= raw-val :remaining-arg-count)
          [(count (:rest navigator)) navigator]

          true
          [raw-val navigator])]
    [[param [real-param offset]] new-navigator]))

(defn- realize-parameter-list [parameter-map navigator]
  (let [[pairs new-navigator]
        (map-passing-context realize-parameter navigator parameter-map)]
    [(into {} pairs) new-navigator]))

(def ^{:private true}
  special-radix-markers {2 "#b" 8 "#o" 16 "#x"})

(def ^{:private true}
  javascript-base-formats {8 "%o", 10 "%d", 16 "%x"})

(def ^{:private true}
     english-cardinal-units
     ["zero" "one" "two" "three" "four" "five" "six" "seven" "eight" "nine"
      "ten" "eleven" "twelve" "thirteen" "fourteen"
      "fifteen" "sixteen" "seventeen" "eighteen" "nineteen"])

(def ^{:private true}
     english-ordinal-units
     ["zeroth" "first" "second" "third" "fourth" "fifth" "sixth" "seventh" "eighth" "ninth"
      "tenth" "eleventh" "twelfth" "thirteenth" "fourteenth"
      "fifteenth" "sixteenth" "seventeenth" "eighteenth" "nineteenth"])

(def ^{:private true}
     english-cardinal-tens
     ["" "" "twenty" "thirty" "forty" "fifty" "sixty" "seventy" "eighty" "ninety"])

(def ^{:private true}
     english-ordinal-tens
     ["" "" "twentieth" "thirtieth" "fortieth" "fiftieth"
      "sixtieth" "seventieth" "eightieth" "ninetieth"])

(def ^{:private true}
     english-scale-numbers
     ["" "thousand" "million" "billion" "trillion" "quadrillion" "quintillion"
      "sextillion" "septillion" "octillion" "nonillion" "decillion"
      "undecillion" "duodecillion" "tredecillion" "quattuordecillion"
      "quindecillion" "sexdecillion" "septendecillion"
      "octodecillion" "novemdecillion" "vigintillion"])

(def ^{:private true}
     old-roman-table
     [[ "I" "II" "III" "IIII" "V" "VI" "VII" "VIII" "VIIII"]
      [ "X" "XX" "XXX" "XXXX" "L" "LX" "LXX" "LXXX" "LXXXX"]
      [ "C" "CC" "CCC" "CCCC" "D" "DC" "DCC" "DCCC" "DCCCC"]
      [ "M" "MM" "MMM"]])

(def ^{:private true}
     new-roman-table
     [[ "I" "II" "III" "IV" "V" "VI" "VII" "VIII" "IX"]
      [ "X" "XX" "XXX" "XL" "L" "LX" "LXX" "LXXX" "XC"]
      [ "C" "CC" "CCC" "CD" "D" "DC" "DCC" "DCCC" "CM"]
      [ "M" "MM" "MMM"]])

(def ^{:private true}
     special-chars {8 "Backspace", 9 "Tab", 10 "Newline", 13 "Return", 32 "Space"})

(defn- abort? [context]
  (let [token (first context)]
    (or (= :up-arrow token) (= :colon-up-arrow token))))

(defn get-pretty-writer
  "Returns the IWriter passed in wrapped in a pretty writer proxy, unless it's
already a pretty writer. Generally, it is unnecessary to call this function, since pprint,
write, and cl-format all call it if they need to. However if you want the state to be
preserved across calls, you will want to wrap them with this.

For example, when you want to generate column-aware output with multiple calls to cl-format,
do it like in this example:

    (defn print-table [aseq column-width]
      (binding [*out* (get-pretty-writer *out*)]
        (doseq [row aseq]
          (doseq [col row]
            (cl-format true \"~4D~7,vT\" col column-width))
          (prn))))

Now when you run:

    user> (print-table (map #(vector % (* % %) (* % % %)) (range 1 11)) 8)

It prints a table of squares and cubes for the numbers from 1 to 10:

       1      1       1
       2      4       8
       3      9      27
       4     16      64
       5     25     125
       6     36     216
       7     49     343
       8     64     512
       9     81     729
      10    100    1000"
  [writer]
  (if (pretty-writer? writer)
    writer
    (pretty-writer writer *print-right-margin* *print-miser-width*)))

(def ^{:private true}
     param-pattern #"^([vV]|#|('.)|([+-]?\d+)|(?=,))")

(def ^{:private true}
     special-params #{:parameter-from-args :remaining-arg-count})

(defn- extract-param [[s offset saw-comma]]
  (let [m (js/RegExp. (.-source param-pattern) "g")
        param (.exec m s)]
    (if param
      (let [token-str (first param)
            remainder (subs s (.-lastIndex m))
            new-offset (+ offset (.-lastIndex m))]
        (if (not (= \, (nth remainder 0)))
          [[token-str offset] [remainder new-offset false]]
          [[token-str offset] [(subs remainder 1) (inc new-offset) true]]))
      (if saw-comma
        (format-error "Badly formed parameters in format directive" offset)
        [nil [s offset]]))))

(defn- extract-params [s offset]
  (consume extract-param [s offset false]))

(defn- translate-param
  "Translate the string representation of a param to the internalized
                                      representation"
  [[p offset]]
  [(cond
     (= (.-length p) 0) nil
     (and (= (.-length p) 1) (contains? #{\v \V} (nth p 0))) :parameter-from-args
     (and (= (.-length p) 1) (= \# (nth p 0))) :remaining-arg-count
     (and (= (.-length p) 2) (= \' (nth p 0))) (nth p 1)
     true (js/parseInt p 10))
   offset])

(def ^{:private true}
     flag-defs {\: :colon, \@ :at})

(defn- extract-flags [s offset]
  (consume
    (fn [[s offset flags]]
      (if (empty? s)
        [nil [s offset flags]]
        (let [flag (get flag-defs (first s))]
          (if flag
            (if (contains? flags flag)
              (format-error
                (str "Flag \"" (first s) "\" appears more than once in a directive")
                offset)
              [true [(subs s 1) (inc offset) (assoc flags flag [true offset])]])
            [nil [s offset flags]]))))
    [s offset {}]))

(defn- check-flags [def flags]
  (let [allowed (:flags def)]
    (if (and (not (:at allowed)) (:at flags))
      (format-error (str "\"@\" is an illegal flag for format directive \"" (:directive def) "\"")
                    (nth (:at flags) 1)))
    (if (and (not (:colon allowed)) (:colon flags))
      (format-error (str "\":\" is an illegal flag for format directive \"" (:directive def) "\"")
                    (nth (:colon flags) 1)))
    (if (and (not (:both allowed)) (:at flags) (:colon flags))
      (format-error (str "Cannot combine \"@\" and \":\" flags for format directive \""
                         (:directive def) "\"")
                    (min (nth (:colon flags) 1) (nth (:at flags) 1))))))

(defn- map-params
  "Takes a directive definition and the list of actual parameters and
a map of flags and returns a map of the parameters and flags with defaults
filled in. We check to make sure that there are the right types and number
of parameters as well."
  [def params flags offset]
  (check-flags def flags)
  (if (> (count params) (count (:params def)))
    (format-error
      (cl-format
        nil
        "Too many parameters for directive \"~C\": ~D~:* ~[were~;was~:;were~] specified but only ~D~:* ~[are~;is~:;are~] allowed"
        (:directive def) (count params) (count (:params def)))
      (second (first params))))
  (doall
    (map #(let [val (first %1)]
           (if (not (or (nil? val) (contains? special-params val)
                        (= (second (second %2)) (type val))))
             (format-error (str "Parameter " (name (first %2))
                                " has bad type in directive \"" (:directive def) "\": "
                                (type val))
                           (second %1))) )
         params (:params def)))

  (merge                                ; create the result map
    (into (array-map) ; start with the default values, make sure the order is right
          (reverse (for [[name [default]] (:params def)] [name [default offset]])))
    (reduce #(apply assoc %1 %2) {} (filter #(first (nth % 1)) (zipmap (keys (:params def)) params))) ; add the specified parameters, filtering out nils
    flags))

(defn- compile-directive [s offset]
  (let [[raw-params [rest offset]] (extract-params s offset)
        [_ [rest offset flags]] (extract-flags rest offset)
        directive (first rest)
        def (get directive-table (string/upper-case directive))
        params (if def (map-params def (map translate-param raw-params) flags offset))]
    (if (not directive)
      (format-error "Format string ended in the middle of a directive" offset))
    (if (not def)
      (format-error (str "Directive \"" directive "\" is undefined") offset))
    [(compiled-directive. ((:generator-fn def) params offset) def params offset)
     (let [remainder (subs rest 1)
           offset (inc offset)
           trim? (and (= \newline (:directive def))
                      (not (:colon params)))
           trim-count (if trim? (prefix-count remainder [\space \tab]) 0)
           remainder (subs remainder trim-count)
           offset (+ offset trim-count)]
       [remainder offset])]))

(defn- compile-raw-string [s offset]
  (compiled-directive. (fn [_ a _] (print s) a) nil {:string s} offset))

(defn- right-bracket [this] (:right (:bracket-info (:def this))))

(defn- separator? [this] (:separator (:bracket-info (:def this))))

(defn- else-separator? [this]
  (and (:separator (:bracket-info (:def this)))
       (:colon (:params this))))

(defn- process-bracket [this remainder]
  (let [[subex remainder] (collect-clauses (:bracket-info (:def this))
                                           (:offset this) remainder)]
    [(compiled-directive.
       (:func this) (:def this)
       (merge (:params this) (tuple-map subex (:offset this)))
       (:offset this))
     remainder]))

(defn- process-clause [bracket-info offset remainder]
  (consume
    (fn [remainder]
      (if (empty? remainder)
        (format-error "No closing bracket found." offset)
        (let [this (first remainder)
              remainder (next remainder)]
          (cond
            (right-bracket this)
            (process-bracket this remainder)

            (= (:right bracket-info) (:directive (:def this)))
            [ nil [:right-bracket (:params this) nil remainder]]

            (else-separator? this)
            [nil [:else nil (:params this) remainder]]

            (separator? this)
            [nil [:separator nil nil remainder]] ;; TODO: check to make sure that there are no params on ~;

            true
            [this remainder]))))
    remainder))

(defn- collect-clauses [bracket-info offset remainder]
  (second
    (consume
      (fn [[clause-map saw-else remainder]]
        (let [[clause [type right-params else-params remainder]]
              (process-clause bracket-info offset remainder)]
          (cond
            (= type :right-bracket)
            [nil [(merge-with concat clause-map
                              {(if saw-else :else :clauses) [clause]
                               :right-params right-params})
                  remainder]]

            (= type :else)
            (cond
              (:else clause-map)
              (format-error "Two else clauses (\"~:;\") inside bracket construction." offset)

              (not (:else bracket-info))
              (format-error "An else clause (\"~:;\") is in a bracket type that doesn't support it."
                            offset)

              (and (= :first (:else bracket-info)) (seq (:clauses clause-map)))
              (format-error
                "The else clause (\"~:;\") is only allowed in the first position for this directive."
                offset)

              true         ; if the ~:; is in the last position, the else clause
              ; is next, this was a regular clause
              (if (= :first (:else bracket-info))
                [true [(merge-with concat clause-map {:else [clause] :else-params else-params})
                       false remainder]]
                [true [(merge-with concat clause-map {:clauses [clause]})
                       true remainder]]))

            (= type :separator)
            (cond
              saw-else
              (format-error "A plain clause (with \"~;\") follows an else clause (\"~:;\") inside bracket construction." offset)

              (not (:allows-separator bracket-info))
              (format-error "A separator (\"~;\") is in a bracket type that doesn't support it."
                            offset)

              true
              [true [(merge-with concat clause-map {:clauses [clause]})
                     false remainder]]))))
      [{:clauses []} false remainder])))

(defn- process-nesting
  "Take a linearly compiled format and process the bracket directives to give it
   the appropriate tree structure"
  [format]
  (first
    (consume
      (fn [remainder]
        (let [this (first remainder)
              remainder (next remainder)
              bracket (:bracket-info (:def this))]
          (if (:right bracket)
            (process-bracket this remainder)
            [this remainder])))
      format)))

(defn- compile-format
  "Compiles format-str into a compiled format which can be used as an argument
to cl-format just like a plain format string. Use this function for improved
performance when you're using the same format string repeatedly"
  [format-str]
  (binding [*format-str* format-str]
    (process-nesting
      (first
        (consume
          (fn [[s offset]]
            (if (empty? s)
              [nil s]
              (let [tilde (.indexOf s \~)]
                (cond
                  (neg? tilde) [(compile-raw-string s offset) ["" (+ offset (.-length s))]]
                  (zero? tilde) (compile-directive (subs s 1) (inc offset))
                  true
                  [(compile-raw-string (subs s 0 tilde) offset) [(subs s tilde) (+ tilde offset)]]))))
          [format-str 0])))))

(defn- needs-pretty
  "determine whether a given compiled format has any directives that depend on the
column number or pretty printing"
  [format]
  (loop [format format]
    (if (empty? format)
      false
      (if (or (:pretty (:flags (:def (first format))))
              (some needs-pretty (first (:clauses (:params (first format)))))
              (some needs-pretty (first (:else (:params (first format))))))
        true
        (recur (next format))))))

(defn- execute-format
  "Executes the format with the arguments."
  {:skip-wiki true}
  ([stream format args]
   (let [sb (StringBuffer.)
         real-stream (if (or (not stream) (true? stream))
                       (StringBufferWriter. sb)
                       stream)
         wrapped-stream (if (and (needs-pretty format)
                                 (not (pretty-writer? real-stream)))
                          (get-pretty-writer real-stream)
                          real-stream)]
     (binding [*out* wrapped-stream]
       (try
         (execute-format format args)
         (finally
           (if-not (identical? real-stream wrapped-stream)
             (-flush wrapped-stream))))
       (cond
         (not stream) (str sb)
         (true? stream) (string-print (str sb))
         :else nil))))
  ([format args]
   (map-passing-context
     (fn [element context]
       (if (abort? context)
         [nil context]
         (let [[params args] (realize-parameter-list
                               (:params element) context)
               [params offsets] (unzip-map params)
               params (assoc params :base-args args)]
           [nil (apply (:func element) [params args offsets])])))
     args
     format)
   nil))

(def ^{:private true} cached-compile (memoize compile-format))

(def ^{:private true} reader-macros
  {'quote "'"
   'var "#'"
   'clojure.core/deref "@",
   'clojure.core/unquote "~"
   'cljs.core/deref "@",
   'cljs.core/unquote "~"})

(def ^{:private true} pprint-array (formatter-out "~<[~;~@{~w~^, ~:_~}~;]~:>"))

(def ^{:private true}
type-map {"core$future_call" "Future",
          "core$promise" "Promise"})

(def ^{:private true} pprint-pqueue (formatter-out "~<<-(~;~@{~w~^ ~_~}~;)-<~:>"))

(defn- brackets
  "Figure out which kind of brackets to use"
  [form]
  (if (vector? form)
    ["[" "]"]
    ["(" ")"]))

(defn- pprint-ns-reference
  "Pretty print a single reference (import, use, etc.) from a namespace decl"
  [reference]
  (if (sequential? reference)
    (let [[start end] (brackets reference)
          [keyw & args] reference]
      (pprint-logical-block :prefix start :suffix end
        ((formatter-out "~w~:i") keyw)
        (loop [args args]
          (when (seq args)
            ((formatter-out " "))
            (let [arg (first args)]
              (if (sequential? arg)
                (let [[start end] (brackets arg)]
                  (pprint-logical-block :prefix start :suffix end
                    (if (and (= (count arg) 3) (keyword? (second arg)))
                      (let [[ns kw lis] arg]
                        ((formatter-out "~w ~w ") ns kw)
                        (if (sequential? lis)
                          ((formatter-out (if (vector? lis)
                                            "~<[~;~@{~w~^ ~:_~}~;]~:>"
                                            "~<(~;~@{~w~^ ~:_~}~;)~:>"))
                            lis)
                          (write-out lis)))
                      (apply (formatter-out "~w ~:i~@{~w~^ ~:_~}") arg)))
                  (when (next args)
                    ((formatter-out "~_"))))
                (do
                  (write-out arg)
                  (when (next args)
                    ((formatter-out "~:_"))))))
            (recur (next args))))))
    (write-out reference)))

(defn- pprint-ns
  "The pretty print dispatch chunk for the ns macro"
  [alis]
  (if (next alis)
    (let [[ns-sym ns-name & stuff] alis
          [doc-str stuff] (if (string? (first stuff))
                            [(first stuff) (next stuff)]
                            [nil stuff])
          [attr-map references] (if (map? (first stuff))
                                  [(first stuff) (next stuff)]
                                  [nil stuff])]
      (pprint-logical-block :prefix "(" :suffix ")"
        ((formatter-out "~w ~1I~@_~w") ns-sym ns-name)
        (when (or doc-str attr-map (seq references))
          ((formatter-out "~@:_")))
        (when doc-str
          (cl-format true "\"~a\"~:[~;~:@_~]" doc-str (or attr-map (seq references))))
        (when attr-map
          ((formatter-out "~w~:[~;~:@_~]") attr-map (seq references)))
        (loop [references references]
          (pprint-ns-reference (first references))
          (when-let [references (next references)]
            (pprint-newline :linear)
            (recur references)))))
    (write-out alis)))

(def ^{:private true} pprint-hold-first (formatter-out "~:<~w~^ ~@_~w~^ ~_~@{~w~^ ~_~}~:>"))

(defn- single-defn [alis has-doc-str?]
  (if (seq alis)
    (do
      (if has-doc-str?
        ((formatter-out " ~_"))
        ((formatter-out " ~@_")))
      ((formatter-out "~{~w~^ ~_~}") alis))))

(defn- multi-defn [alis has-doc-str?]
  (if (seq alis)
    ((formatter-out " ~_~{~w~^ ~_~}") alis)))

(defn- pprint-defn [alis]
  (if (next alis)
    (let [[defn-sym defn-name & stuff] alis
          [doc-str stuff] (if (string? (first stuff))
                            [(first stuff) (next stuff)]
                            [nil stuff])
          [attr-map stuff] (if (map? (first stuff))
                             [(first stuff) (next stuff)]
                             [nil stuff])]
      (pprint-logical-block :prefix "(" :suffix ")"
        ((formatter-out "~w ~1I~@_~w") defn-sym defn-name)
        (if doc-str
          ((formatter-out " ~_~w") doc-str))
        (if attr-map
          ((formatter-out " ~_~w") attr-map))
        ;; Note: the multi-defn case will work OK for malformed defns too
        (cond
          (vector? (first stuff)) (single-defn stuff (or doc-str attr-map))
          :else (multi-defn stuff (or doc-str attr-map)))))
    (pprint-simple-code-list alis)))

(defn- pprint-binding-form [binding-vec]
  (pprint-logical-block :prefix "[" :suffix "]"
    (print-length-loop [binding binding-vec]
      (when (seq binding)
        (pprint-logical-block binding
          (write-out (first binding))
          (when (next binding)
            (-write *out* " ")
            (pprint-newline :miser)
            (write-out (second binding))))
        (when (next (rest binding))
          (-write *out* " ")
          (pprint-newline :linear)
          (recur (next (rest binding))))))))

(defn- pprint-let [alis]
  (let [base-sym (first alis)]
    (pprint-logical-block :prefix "(" :suffix ")"
      (if (and (next alis) (vector? (second alis)))
        (do
          ((formatter-out "~w ~1I~@_") base-sym)
          (pprint-binding-form (second alis))
          ((formatter-out " ~_~{~w~^ ~_~}") (next (rest alis))))
        (pprint-simple-code-list alis)))))

(def ^{:private true} pprint-if (formatter-out "~:<~1I~w~^ ~@_~w~@{ ~_~w~}~:>"))

(defn- pprint-cond [alis]
  (pprint-logical-block :prefix "(" :suffix ")"
    (pprint-indent :block 1)
    (write-out (first alis))
    (when (next alis)
      (-write *out* " ")
      (pprint-newline :linear)
      (print-length-loop [alis (next alis)]
        (when alis
          (pprint-logical-block alis
            (write-out (first alis))
            (when (next alis)
              (-write *out* " ")
              (pprint-newline :miser)
              (write-out (second alis))))
          (when (next (rest alis))
            (-write *out* " ")
            (pprint-newline :linear)
            (recur (next (rest alis)))))))))

(defn- pprint-condp [alis]
  (if (> (count alis) 3)
    (pprint-logical-block :prefix "(" :suffix ")"
      (pprint-indent :block 1)
      (apply (formatter-out "~w ~@_~w ~@_~w ~_") alis)
      (print-length-loop [alis (seq (drop 3 alis))]
        (when alis
          (pprint-logical-block alis
            (write-out (first alis))
            (when (next alis)
              (-write *out* " ")
              (pprint-newline :miser)
              (write-out (second alis))))
          (when (next (rest alis))
            (-write *out* " ")
            (pprint-newline :linear)
            (recur (next (rest alis)))))))
    (pprint-simple-code-list alis)))

(def ^:dynamic ^{:private true} *symbol-map* {})

(defn- pprint-anon-func [alis]
  (let [args (second alis)
        nlis (first (rest (rest alis)))]
    (if (vector? args)
      (binding [*symbol-map* (if (= 1 (count args))
                               {(first args) "%"}
                               (into {}
                                 (map
                                   #(vector %1 (str \% %2))
                                   args
                                   (range 1 (inc (count args))))))]
        ((formatter-out "~<#(~;~@{~w~^ ~_~}~;)~:>") nlis))
      (pprint-simple-code-list alis))))

(defn- pprint-simple-code-list [alis]
  (pprint-logical-block :prefix "(" :suffix ")"
    (pprint-indent :block 1)
    (print-length-loop [alis (seq alis)]
      (when alis
        (write-out (first alis))
        (when (next alis)
          (-write *out* " ")
          (pprint-newline :linear)
          (recur (next alis)))))))

(defn- two-forms [amap]
  (into {}
    (mapcat
      identity
      (for [x amap]
        [x [(symbol (name (first x))) (second x)]]))))

(defn- add-core-ns [amap]
  (let [core "clojure.core"]
    (into {}
      (map #(let [[s f] %]
             (if (not (or (namespace s) (special-symbol? s)))
               [(symbol core (name s)) f]
               %))
        amap))))

(def ^:dynamic ^{:private true} *code-table*
  (two-forms
    (add-core-ns
      {'def pprint-hold-first, 'defonce pprint-hold-first,
       'defn pprint-defn, 'defn- pprint-defn, 'defmacro pprint-defn, 'fn pprint-defn,
       'let pprint-let, 'loop pprint-let, 'binding pprint-let,
       'with-local-vars pprint-let, 'with-open pprint-let, 'when-let pprint-let,
       'if-let pprint-let, 'doseq pprint-let, 'dotimes pprint-let,
       'when-first pprint-let,
       'if pprint-if, 'if-not pprint-if, 'when pprint-if, 'when-not pprint-if,
       'cond pprint-cond, 'condp pprint-condp,
       'fn* pprint-anon-func,
       '. pprint-hold-first, '.. pprint-hold-first, '-> pprint-hold-first,
       'locking pprint-hold-first, 'struct pprint-hold-first,
       'struct-map pprint-hold-first, 'ns pprint-ns
       })))

(defn- add-padding [width s]
  (let [padding (max 0 (- width (count s)))]
    (apply str (clojure.string/join (repeat padding \space)) s)))

(defn print-table
  "Prints a collection of maps in a textual table. Prints table headings
   ks, and then a line of output for each row, corresponding to the keys
   in ks. If ks are not specified, use the keys of the first item in rows."
  {:added "1.3"}
  ([ks rows]
   (when (seq rows)
     (let [widths  (map
                     (fn [k]
                       (apply max (count (str k)) (map #(count (str (get % k))) rows)))
                     ks)
           spacers (map #(apply str (repeat % "-")) widths)
           fmt-row (fn [leader divider trailer row]
                     (str leader
                       (apply str (interpose divider
                                    (for [[col width] (map vector (map #(get row %) ks) widths)]
                                      (add-padding width (str col)))))
                       trailer))]
       (cljs.core/println)
       (cljs.core/println (fmt-row "| " " | " " |" (zipmap ks ks)))
       (cljs.core/println (fmt-row "|-" "-+-" "-|" (zipmap ks spacers)))
       (doseq [row rows]
         (cljs.core/println (fmt-row "| " " | " " |" row))))))
  ([rows] (print-table (keys (first rows)) rows)))
