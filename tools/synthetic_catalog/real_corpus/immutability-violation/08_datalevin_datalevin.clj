(ns ^:no-doc datalevin.hu
  "Fast encoder and decoder for Hu-Tucker codes. Used for key compression."
  (:require
   [datalevin.util :as u])
  (:import
   [java.util LinkedList Arrays]
   [java.nio ByteBuffer ByteOrder]
   [java.io DataOutputStream BufferedOutputStream FileOutputStream
    DataInputStream BufferedInputStream InputStream]
   [org.eclipse.collections.impl.map.mutable.primitive LongObjectHashMap]
   [org.eclipse.collections.impl.map.mutable.primitive ObjectLongHashMap]
   [datalevin.utl LeftistHeap BitOps]))

(defprotocol INode
  (leaf? [_])
  (left-child [_])
  (right-child [_])
  (set-left-child [_ node])
  (set-right-child [_ node]))

(deftype SeqNode [^long sum          ;; sum of min freq and 2nd min freq
                  ^int i             ;; idx of min freq TreeNode
                  ^int j             ;; idx of 2nd min freq TreeNode
                  ^int l             ;; idx of the left terminal TreeNode
                  ^int r             ;; idx of the right terminal TreeNode
                  ^LeftistHeap heap] ;; heap of TreeNodes in this seq
  Object
  (hashCode [_] l)
  (equals [_ other] (= l (.-l ^SeqNode other))))

(defn- master-pq
  []
  (proxy [LeftistHeap] []
    (lessThan [^SeqNode a ^SeqNode b]
      (< (long (u/combine-cmp
                 (compare ^long (.-sum a) ^long (.-sum b))
                 (compare ^int (.-l a) ^int (.-l b)))) ;; tie breaker
         0))))

(defprotocol ITreeNode
  (left-seq [_])
  (right-seq [_])
  (set-left-seq [_ s])
  (set-right-seq [_ s]))

(deftype TreeNode [^int idx
                   ^long freq
                   ^:unsynchronized-mutable ^SeqNode left-seq
                   ^:unsynchronized-mutable ^SeqNode right-seq
                   left-child
                   right-child]
  INode
  (leaf? [_] (nil? left-child))

  ITreeNode
  (left-seq [_] left-seq)
  (right-seq [_] right-seq)
  (set-left-seq [_ s] (set! left-seq s))
  (set-right-seq [_ s] (set! right-seq s))

  Object
  (hashCode [_] idx)
  (equals [_ other] (= idx (.-idx ^TreeNode other))))

(defn- huffman-pq
  []
  (proxy [LeftistHeap] []
    (lessThan [^TreeNode a ^TreeNode b]
      (< (long (u/combine-cmp
                 (compare ^long (.-freq a) ^long (.-freq b))
                 (compare ^int (.-idx a) ^int (.-idx b))))
         0))))

(defn- init-queues
  [n ^longs freqs ^"[Ldatalevin.hu.TreeNode;" work
   ^"[Ldatalevin.hu.TreeNode;" terminals ^LeftistHeap mpq]
  (dotimes [k n]
    (let [node (TreeNode. k (aget freqs k) nil nil nil nil)]
      (aset terminals k node)
      (aset work k node)))
  (dotimes [k (dec ^long n)]
    (let [k+1  (inc k)
          t    ^TreeNode (aget work k)
          t+1  ^TreeNode (aget work k+1)
          wk   (.-freq t)
          wk+1 (.-freq t+1)
          i    (if (<= wk wk+1) k k+1)
          j    (if (= i k) k+1 k)
          hpq  ^LeftistHeap (huffman-pq)
          sn   (SeqNode. (+ wk wk+1) i j k k+1 hpq)]
      (set-right-seq t sn)
      (set-left-seq t+1 sn)
      (.insert hpq t)
      (.insert hpq t+1)
      (.insert mpq sn))))

(defn- build-level-tree
  [^long n ^longs freqs]
  (let [^"[Ldatalevin.hu.TreeNode;" work      (make-array TreeNode n)
        ^"[Ldatalevin.hu.TreeNode;" terminals (make-array TreeNode n)
        ^LeftistHeap mpq                      (master-pq)]
    (init-queues n freqs work terminals mpq)
    (dotimes [_ (dec n)]
      (let [m  ^SeqNode (.findMin mpq)
            i  (.-i m)
            j  (.-j m)
            l  (if (<= i j) i j)
            r  (if (= l i) j i)
            nl (aget work l)
            nr (aget work r)
            nn (TreeNode. l (.-sum m) nil nil nl nr)]
        (aset work l nn)
        (aset work r nil)
        (cond
          ;; combine 2 terminal nodes, need to merge 3 or 2 seqs
          (and (leaf? nl) (leaf? nr))
          (let [tl     (aget terminals l)
                tr     (aget terminals r)
                ll-seq ^SeqNode (left-seq tl)
                l      (if ll-seq (.-l ll-seq) -1)
                ll-hpq (when ll-seq
                         (doto ^LeftistHeap (.-heap ll-seq)
                           (.deleteElement tl)))
                lr-hpq (doto ^LeftistHeap (.-heap ^SeqNode (right-seq tl))
                         (.deleteElement tl))
                _      (doto ^LeftistHeap (.-heap ^SeqNode (left-seq tr))
                         (.deleteElement tr))
                rr-seq ^SeqNode (right-seq tr)
                r      (if rr-seq (.r rr-seq) n)
                rr-hpq (when rr-seq
                         (doto ^LeftistHeap (.-heap rr-seq)
                           (.deleteElement tr)))
                n-hpq  ^LeftistHeap (doto lr-hpq
                                      (.merge ll-hpq) (.merge rr-hpq)
                                      (.insert nn))
                minn   ^TreeNode (.findMin n-hpq)
                minn1  ^TreeNode (.findNextMin n-hpq)
                sn     (SeqNode. (+ (.-freq minn) (.-freq minn1))
                                 (.-idx minn) (.-idx minn1) l r n-hpq)]
            (when-not (= l -1) (set-right-seq (aget terminals l) sn))
            (when-not (= r n) (set-left-seq (aget terminals r) sn))
            (doto mpq
              (.deleteElement ll-seq) (.deleteElement rr-seq)
              (.deleteMin) (.insert sn)))
          ;; combine 2 internal nodes, no seq merge needed
          (and (not (leaf? nl)) (not (leaf? nr)))
          (let [hpq   ^LeftistHeap (.-heap m)
                n-hpq (doto hpq
                        (.deleteMin) (.deleteMin) (.insert nn))
                minn  ^TreeNode (.findMin n-hpq)
                minn1 ^TreeNode (.findNextMin n-hpq)
                l     (.-l m)
                r     (.-r m)
                sn    (when minn1
                        (SeqNode. (+ (.-freq minn) (.-freq minn1))
                                  (.-idx minn) (.-idx minn1) l r n-hpq))]
            (when-not (= l -1) (set-right-seq (aget terminals l) sn))
            (when-not (= r n) (set-left-seq (aget terminals r) sn))
            (doto mpq (.deleteMin) (.insert sn)))
          ;; combine a terminal and an internal node, need to merge two seqs
          :else
          (let [hpq   ^LeftistHeap (.-heap m)
                t     ^TreeNode (if (leaf? nl) nl nr)
                l-seq ^SeqNode (left-seq t)
                l     (if l-seq (.l l-seq) -1)
                l-hpq (when l-seq ^LeftistHeap (.-heap l-seq))
                r-seq ^SeqNode (right-seq t)
                r     (if r-seq (.r r-seq) n)
                r-hpq (when r-seq ^LeftistHeap (.-heap r-seq))
                o-hpq ^SeqNode (if (= hpq l-hpq) r-hpq l-hpq)
                o-seq (if (= o-hpq r-hpq) r-seq l-seq)
                _     (when o-hpq (.deleteElement o-hpq t))
                n-hpq (doto hpq
                        (.deleteMin) (.deleteMin)
                        (.merge o-hpq) (.insert nn))
                minn  ^TreeNode (.findMin n-hpq)
                minn1 ^TreeNode (.findNextMin n-hpq)
                sn    (SeqNode. (+ (.-freq minn) (.-freq minn1))
                                (.-idx minn) (.-idx minn1) l r n-hpq)]
            (when-not (= l -1) (set-right-seq (aget terminals l) sn))
            (when-not (= r n) (set-left-seq (aget terminals r) sn))
            (doto mpq
              (.deleteMin) (.deleteElement o-seq) (.insert sn))))))
    (aget work 0)))

(defn create-levels
  [^long n ^longs freqs]
  (let [tree   (build-level-tree n freqs)
        levels (byte-array n)]
    (letfn [(traverse [^TreeNode node ^long level]
              (if (leaf? node)
                (aset levels (.-idx node) (byte level))
                (let [l+1 (inc level)]
                  (traverse (.-left-child node) l+1)
                  (traverse (.-right-child node) l+1))))]
      (traverse tree 0)
      levels)))

(deftype Node [level sym left-child right-child]
  INode
  (leaf? [_] (nil? left-child)))

(defn- build-code-tree
  [^long n ^bytes levels]
  (let [cur   (volatile! 0)
        stack (LinkedList.)]
    (while (not (and (= n @cur) (= 1 (.size stack))))
      (if (and (<= 2 (.size stack)) (= (.-level ^Node (.get stack 0))
                                       (.-level ^Node (.get stack 1))))
        (let [top   ^Node (.pop stack)
              top-1 ^Node (.pop stack)
              level (dec ^byte (.-level top))]
          (.push stack (Node. level nil top-1 top)))
        (when (< ^long @cur n)
          (let [sym   @cur
                level (aget levels sym)]
            (.push stack (Node. level sym nil nil))
            (vswap! cur u/long-inc)))))
    (.pop stack)))

(defn create-codes
  [^long n ^bytes lens ^ints codes ^longs freqs]
  (let [levels (create-levels n freqs)
        root   (build-code-tree n levels)]
    (letfn [(traverse [^Node node ^long code]
              (if (leaf? node)
                (let [sym (.-sym node)]
                  (aset lens sym (byte (.-level node)))
                  (aset codes sym (int code)))
                (let [code1 (bit-shift-left code 1)]
                  (traverse (.-left-child node) code1)
                  (traverse (.-right-child node) (inc code1)))))]
      (traverse root 0))))
