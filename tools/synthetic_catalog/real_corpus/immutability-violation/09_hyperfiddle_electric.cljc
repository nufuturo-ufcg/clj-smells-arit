(ns hyperfiddle.electric.impl.runtime3
  (:refer-clojure :exclude [resolve *e])
  (:require [hyperfiddle.incseq :as i]
            [hyperfiddle.incseq.perm-impl :as p]
            [hyperfiddle.incseq.arrays-impl :as a]
            [missionary.core :as m]
            #?(:cljs missionary.impl.Propagator)
            [clojure.pprint]
            #?(:clj [clojure.tools.logging :as log])
            [hyperfiddle.electric.impl.lang3 :as-alias lang]
            [cognitect.transit :as t]
            [hyperfiddle.incseq.diff-impl :as d]
            [hyperfiddle.incseq.stateful-diff-impl :as sd]
            [hyperfiddle.incseq.to-stateful-impl :as ts]
            [hyperfiddle.incseq.from-stateful-impl :as fs]
            [clojure.string :as str]
            [hyperfiddle.electric.impl.dynamic-local :as dl]
            [hyperfiddle.electric.impl.missionary-util :as mu]
            [hyperfiddle.incseq.sync-impl :refer [get-sync cas-sync]])
  (:import missionary.Cancelled
           missionary.impl.PairingHeap
           #?(:clj missionary.impl.Propagator$Publisher)
           #?(:clj (clojure.lang IFn IDeref))
           #?(:clj (java.io ByteArrayInputStream ByteArrayOutputStream Writer))
           #?(:clj (java.util.concurrent.atomic AtomicInteger AtomicReference))))

(defn run
  ([f] (f) f)
  ([f x] (f x) f)
  ([f x y] (f x y) f)
  ([f x y & zs] (apply f x y zs) f))

(deftype Peer [site defs request next-id])

(defn peer-site [^Peer peer]
  (.-site peer))

(deftype Signal [slot site deps meta input request flow])

(defn signal-site [^Signal signal]
  (.-site signal))

(defn signal-deps [^Signal signal]
  (.-deps signal))

(defn signal-flow [^Signal signal]
  (.-flow signal))

(def session-slot-socket 0)

(def session-slot-signal 1)

(def session-slot-request 2)

(def session-slot-flags 3)

(def session-slot-delay 4)

(def session-slot-event 5)

(def session-slot-buffer 6)

(def session-slot-size 7)

(def session-slot-updated 9)

(def session-slots 10)

(def socket-slot-root 0)

(def socket-slot-step 1)

(def socket-slot-done 2)

(def socket-slot-head 6)

(def socket-slot-sync 7)

(def socket-slot-owner 8)

(def socket-slot-pending 9)

(def socket-slot-sessions 10)

(def socket-slot-updated 12)

(def socket-slot-message-frames 20)

(def event-slot-type 0)

(def event-slot-target 1)

(def event-slot-process 2)

(def event-slot-child 3)

(def event-slot-sibling 4)

(def event-slots 5)

(def item-slot-parent 0)

(def item-slot-current 1)

(def item-slot-position 2)

(def item-slot-event 3)

(def request-flag-local 1)

(def request-flags 3)

(defprotocol Expr
  (deps [_ rf r]) ;; emits signals
  (t [_])
  (peephole [_])
  (pp [_]))

(deftype Cdef [frees nodes calls result build])

(def cdef ->Cdef)

(deftype Frame [peer slot id site ctor ^objects nodes ^objects tags
                ^:unsynchronized-mutable ^:mutable hash-memo]
  #?(:clj Object)
  #?(:cljs IHash)
  (#?(:clj hashCode :cljs -hash) [_]
    (if-some [h hash-memo]
      h (set! hash-memo
          (-> (hash Frame)
            (hash-combine (hash peer))
            (hash-combine (hash site))
            (hash-combine (hash id))))))
  #?(:cljs IEquiv)
  (#?(:clj equals :cljs -equiv) [_ other]
    (and (instance? Frame other)
      (= peer (.-peer ^Frame other))
      (= site (.-site ^Frame other))
      (= id (.-id ^Frame other))))
  IFn
  (#?(:clj invoke :cljs -invoke) [this step done]
    ((frame-result this) step done)))

(deftype Tag [frame index
              ^:unsynchronized-mutable ^:mutable hash-memo]
  #?(:clj Object)
  #?(:cljs IHash)
  (#?(:clj hashCode :cljs -hash) [_]
    (if-some [h hash-memo]
      h (set! hash-memo (-> (hash Tag)
                          (hash-combine (hash frame))
                          (hash-combine (hash index))))))
  #?(:cljs IEquiv)
  (#?(:clj equals :cljs -equiv) [_ other]
    (and (instance? Tag other)
      (= frame (.-frame ^Tag other))
      (= index (.-index ^Tag other)))))

(defn tag [^Frame frame id]
  (->Tag frame id nil))

(defn frame-ctor
  "Returns the constructor of given frame."
  [^Frame frame]
  (.-ctor frame))

(defn frame-peer
  "Returns the peer of given frame."
  [^Frame frame]
  (.-peer frame))

(defn frame-signal [^Frame frame id]
  (if (neg? id)
    (aget ^objects (.-nodes frame) (- -1 id))
    (aget ^objects (.-tags frame) id)))

(deftype Slot [^Frame frame id]
  #?(:clj Object)
  #?(:cljs IHash)
  (#?(:clj hashCode :cljs -hash) [_]
    (hash (frame-signal frame id)))
  #?(:cljs IEquiv)
  (#?(:clj equals :cljs -equiv) [_ that]
    (and (instance? Slot that)
      (= (frame-signal frame id)
        (frame-signal (.-frame ^Slot that) (.-id ^Slot that)))))
  Expr
  (deps [_ rf r] (rf r (frame-signal frame id)))
  (t [_] ::slot)
  (peephole [this] this)
  (pp [_] (list 'Slot id))
  IFn
  (#?(:clj invoke :cljs -invoke) [_ step done]
    ((signal-flow (frame-signal frame id)) step done)))

(defn signal-slot
  {:tag Slot}
  [^Signal signal]
  (.-slot signal))

(defn slot-frame
  "Returns the frame of given slot."
  {:tag Frame}
  [^Slot slot]
  (.-frame slot))

(defn session-signal ^Signal [^objects session]
  (aget session session-slot-signal))

(defn item-parent ^objects [^objects item]
  (aget item item-slot-parent))

(defn event-cancel [^objects event]
  ((aget event event-slot-process)))

(defn get-child [_ ^objects event]
  (aget event event-slot-child))

(defn set-child [_ ^objects event x]
  (aset event event-slot-child x))

(defn get-sibling [_ ^objects event]
  (aget event event-slot-sibling))

(defn set-sibling [_ ^objects event x]
  (aset event event-slot-sibling x))

(defn event-priority [^objects event]
  (case (aget event event-slot-type)
    :message 0
    :request 1
    (:output :item) 2))

(defn event-publisher [^objects event]
  (let [target (aget event event-slot-target)]
    (case (aget event event-slot-type)
      :output (signal-flow (session-signal target))
      :item (signal-flow (session-signal (item-parent target))))))

(def impl
  (PairingHeap/impl
    (fn [_ ^objects x ^objects y]
      (case (compare (event-priority x) (event-priority y))
        -1 true
        0 (case (compare (event-publisher x) (event-publisher y))
            -1 false
            0 (= :output (aget x event-slot-type))
            +1 true)
        +1 false))
    get-child set-child get-sibling set-sibling))

(def idle #?(:clj (Object.) :cljs (js-obj)))

(defn event-terminated [event]
  (identical? idle (get-child nil event)))

(defn socket-event [^objects socket event]
  (aset socket socket-slot-pending
    (dec (aget socket socket-slot-pending)))
  (set-sibling socket event nil)
  (when-not ^boolean (event-terminated event)
    (aset socket socket-slot-head
      (if-some [h (aget socket socket-slot-head)]
        (PairingHeap/meld impl nil h event)
        event)) nil))

(defn socket-consume [^objects socket event]
  (loop [event event]
    (let [s (get-sibling socket event)]
      (socket-event socket event)
      (when (some? s) (recur s)))))

(defn socket-dequeue [^objects socket]
  (aset socket socket-slot-pending (inc (aget socket socket-slot-pending)))
  (let [h (aget socket socket-slot-head)]
    (aset socket socket-slot-head (PairingHeap/dmin impl nil h))
    (set-sibling socket h idle) h))

(defn socket-claim [^objects socket]
  (aset socket socket-slot-owner #?(:clj (Thread/currentThread) :cljs true)))

(defn socket-yield [^objects socket]
  (aset socket socket-slot-owner #?(:clj nil :cljs false)))

(defn socket-terminate [^objects socket ^objects event]
  (set-child socket event idle)
  (socket-event socket event))

(defn socket-flush [^objects socket ^objects event]
  (socket-claim socket)
  (try @(aget event event-slot-process)
       (catch #?(:clj Throwable :cljs :default) _))
  (socket-yield socket))

(defn socket-ready [^objects socket]
  (loop []
    (if (nil? (aget socket socket-slot-head))
      (if (zero? (aget socket socket-slot-pending))
        ((aget socket socket-slot-done))
        (cas-sync socket socket-slot-sync nil idle
          nil (do (loop []
                    (let [s (get-sync socket socket-slot-sync)]
                      (cas-sync socket socket-slot-sync s nil
                        (socket-consume socket s)
                        (recur))))
                  (recur))))
      (if-some [step (aget socket socket-slot-step)]
        (step)
        (let [^objects event (socket-dequeue socket)]
          (case (aget event event-slot-type)
            :message (socket-flush socket event)
            :request (socket-flush socket event)
            :output (let [^objects target (aget event event-slot-target)]
                      (if (nil? (aget target session-slot-event))
                        (socket-terminate socket event)
                        (socket-flush socket event)))
            :item (let [^objects target (aget event event-slot-target)]
                    (if (nil? (aget target item-slot-event))
                      (socket-terminate socket event)
                      (socket-flush socket event))))
          (recur))))))

(def consecutive #{:output :item})

(defn socket-step [^objects socket event]
  (if (and #_(contains? consecutive (aget event event-slot-type))
        #?(:clj  (identical? (Thread/currentThread) (aget socket socket-slot-owner))
           :cljs ^boolean (aget socket socket-slot-owner)))
    (socket-event socket event)
    (loop []
      (let [s (get-sync socket socket-slot-sync)]
        (if (identical? s idle)
          (cas-sync socket socket-slot-sync idle nil
            (do (socket-event socket event)
                (socket-ready socket)) (recur))
          (do (set-sibling socket event s)
              (cas-sync socket socket-slot-sync s event
                nil (recur))))))))

(defn node
  "Returns the signal node id for given frame."
  {:tag Slot}
  [^Frame frame id]
  (->Slot frame (- -1 id)))

(defn frame-result-slot [^Frame frame]
  (let [^objects nodes (.-nodes frame)]
    (node frame (dec (alength nodes)))))

(defn frame-result-signal [^Frame frame]
  (let [^objects nodes (.-nodes frame)]
    (aget nodes (dec (alength nodes)))))

(defn call? [^Slot slot]
  (not (neg? (.-id slot))))

(defn signal-local? [^Signal signal]
  (= (signal-site signal) (peer-site (frame-peer (slot-frame (signal-slot signal))))))

(defn request-flag ^long [^longs request flag]
  (if (zero? (aget request flag))
    0 (bit-shift-left 1 flag)))

(defn request-state ^long [^longs request]
  (-> (request-flag request 0)
    (bit-or (request-flag request 1))
    (bit-or (request-flag request 2))))

(defn session-spawn [^objects socket signal ^longs request delay]
  (let [session (object-array session-slots)
        ^objects event (if (= (signal-site signal) (peer-site (frame-peer (aget socket socket-slot-root))))
                         (doto (object-array event-slots)
                           (aset event-slot-type :output)
                           (aset event-slot-target session))
                         (m/store d/combine (d/empty-diff 0)))]
    (aset socket socket-slot-sessions (assoc (aget socket socket-slot-sessions) (signal-slot signal) session))
    (aset session session-slot-socket socket)
    (aset session session-slot-signal signal)
    (aset session session-slot-request request)
    (aset session session-slot-flags ^Object (request-state request))
    (aset session session-slot-delay delay)
    (aset session session-slot-buffer (object-array 1))
    (aset session session-slot-size (+))
    (aset session session-slot-event event)
    (aset session session-slot-updated session)
    (if (signal-local? signal)
      (do (aset socket socket-slot-pending
            (inc (aget socket socket-slot-pending)))
          (aset event event-slot-process
            ((fs/flow (signal-flow signal))
             #(socket-step socket event)
             #(do (aset session session-slot-event nil)
                  (socket-step socket event)))))
      ((.-input ^Signal signal) (ts/flow event)))
    session))

(defn session-check-create ^objects [^objects socket ^Signal signal]
  (let [slot (signal-slot signal)]
    (if-some [session (get (aget socket socket-slot-sessions) slot)]
      session (session-spawn socket signal (a/long-array request-flags) 0))))

(defn walk [^objects socket remote? diff flag signal]
  (if (or #_(call? (signal-slot signal)) (= remote? (signal-local? signal)))
    (let [^objects session (session-check-create socket signal)
          ^longs request (aget session session-slot-request)
          prev (aget request flag)
          curr (aset request flag ^long (unchecked-add prev diff))]
      (when-not (= (zero? prev) (zero? curr))
        #_
        (prn :request-toggle (signal-slot signal)
          (case flag 0 :ack 1 :local 2 :remote)
          (if (zero? prev) :up :down))
        (when (identical? session (aget session session-slot-updated))
          (aset session session-slot-updated (aget socket socket-slot-updated))
          (aset socket socket-slot-updated session))
        (reduce run (partial walk socket (not remote?) diff flag) (signal-deps signal))))
    (reduce run (partial walk socket remote? diff flag) (signal-deps signal))))

(defn socket-local-frame-down [^objects socket frame]
  #_#_
  (walk socket false -1 request-flag-local (frame-result-signal frame))
  (aset socket socket-slot-message-frames
    (assoc! (aget socket socket-slot-message-frames) frame -1)))

(defn output-item-dispose [^objects session pos]
  (let [^objects socket (aget session session-slot-socket)
        ^objects buffer (aget session session-slot-buffer)
        ^Signal signal (aget session session-slot-signal)
        ^Slot slot (.-slot signal)
        ^objects item (aget buffer pos)
        x (aget item item-slot-current)]
    (aset item item-slot-position nil)
    (aset buffer pos nil)
    (when-some [event (aget item item-slot-event)]
      (event-cancel event))
    (when (call? slot) (socket-local-frame-down socket x))))

(defn slot-toggle [^Slot slot]
  (let [^Peer peer (.-peer (slot-frame slot))]
    ((.-request peer) #{slot})))

(defn signal-up [^Signal signal]
  (when (zero? #?(:clj  (.getAndIncrement ^AtomicInteger (.-request signal))
                  :cljs (let [n (.-request signal)] (set! (.-request signal) (inc n)) n)))
    (slot-toggle (signal-slot signal))))

(defn signal-down [^Signal signal]
  (when (zero? #?(:clj  (.decrementAndGet ^AtomicInteger (.-request signal))
                  :cljs (let [n (dec (.-request signal))] (set! (.-request signal) n) n)))
    (slot-toggle (signal-slot signal))))

(defn incseq
  ([expr]
   (m/sample {}
     (m/observe
       (fn [!] (! nil)
         (deps expr run signal-up)
         #(deps expr run signal-down)))
     expr))
  ;; compat, TODO change compiler
  ([_ expr] (incseq expr)))

(defn ctor
  "Returns the constructor for cdef coordinates key and idx, with given free variables."
  [key idx & frees] [key idx (vec frees) {}])

(defn free
  "Returns the free variable id for given frame."
  {:tag Slot}
  [^Frame frame id]
  (let [[_ _ free _] (frame-ctor frame)]
    (free id)))

(defn frame-result [frame]
  (incseq (frame-result-slot frame)))
