(ns ^:no-doc datascript.query
  (:require
    [#?(:cljs cljs.reader :clj clojure.edn) :as edn]
    [clojure.set :as set]
    [clojure.string :as str]
    [clojure.walk :as walk]
    [datascript.built-ins :as built-ins]
    [datascript.db :as db]
    [me.tonsky.persistent-sorted-set.arrays :as da]
    [datascript.lru :as lru]
    [datascript.impl.entity :as de]
    [datascript.parser :as dp #?@(:cljs [:refer [BindColl BindIgnore BindScalar BindTuple Constant
                                                 FindColl FindRel FindScalar FindTuple PlainSymbol
                                                 RulesVar SrcVar Variable]])]
    [datascript.pull-api :as dpa]
    [datascript.util :as util])
  #?(:clj
     (:import
       [clojure.lang ILookup LazilyPersistentVector]
       [datascript.parser BindColl BindIgnore BindScalar BindTuple
        Constant FindColl FindRel FindScalar FindTuple PlainSymbol
        RulesVar SrcVar Variable])))

(defrecord Relation [attrs tuples])

(defn intersect-keys [attrs1 attrs2]
  (set/intersection (set (keys attrs1))
    (set (keys attrs2))))

(defn getter-fn [attrs attr]
  (let [idx (attrs attr)]
    (if (contains? *lookup-attrs* attr)
      (if (int? idx)
        (let [idx (int idx)]
          (fn contained-int-getter-fn [tuple]
            (let [eid #?(:cljs (da/aget tuple idx)
                         :clj (if (.isArray (.getClass ^Object tuple))
                                (aget ^objects tuple idx)
                                (nth tuple idx)))]
              (cond
                (number? eid)     eid ;; quick path to avoid fn call
                (sequential? eid) (db/entid *implicit-source* eid)
                (da/array? eid)   (db/entid *implicit-source* eid)
                :else             eid))))
        ;; If the index is not an int?, the target can never be an array
        (fn contained-getter-fn [tuple]
          (let [eid #?(:cljs (da/aget tuple idx)
                       :clj (.valAt ^ILookup tuple idx))]
            (cond
              (number? eid)     eid ;; quick path to avoid fn call
              (sequential? eid) (db/entid *implicit-source* eid)
              (da/array? eid)   (db/entid *implicit-source* eid)
              :else             eid))))
      (if (int? idx)
        (let [idx (int idx)]
          (fn int-getter [tuple]
            #?(:cljs (da/aget tuple idx)
               :clj (if (.isArray (.getClass ^Object tuple))
                      (aget ^objects tuple idx)
                      (nth tuple idx)))))
        ;; If the index is not an int?, the target can never be an array
        (fn getter [tuple]
          #?(:cljs (da/aget tuple idx)
             :clj (.valAt ^ILookup tuple idx)))))))

(defn tuple-key-fn
  [attrs common-attrs]
  (let [n (count common-attrs)]
    (if (== n 1)
      (getter-fn attrs (first common-attrs))
      (let [^objects getters-arr #?(:clj (into-array Object common-attrs)
                                    :cljs (into-array common-attrs))]
        (loop [i 0]
          (if (< i n)
            (do
              (aset getters-arr i (getter-fn attrs (aget getters-arr i)))
              (recur (unchecked-inc i)))
            #?(:clj
               (fn [tuple]
                 (let [^objects arr (make-array Object n)]
                   (loop [i 0]
                     (if (< i n)
                       (do
                         (aset arr i ((aget getters-arr i) tuple))
                         (recur (unchecked-inc i)))
                       (LazilyPersistentVector/createOwning arr)))))
               :cljs (fn [tuple]
                       (list* (.map getters-arr #(% tuple)))))))))))

(defn -group-by
  [f init coll]
  (persistent!
    (reduce
      (fn [ret x]
        (let [k (f x)]
          (assoc! ret k (conj (get ret k init) x))))
      (transient {}) coll)))

(defn hash-attrs [key-fn tuples]
  (-group-by key-fn '() tuples))

(defn hash-join [rel1 rel2]
  (let [tuples1       (:tuples rel1)
        tuples2       (:tuples rel2)
        attrs1        (:attrs rel1)
        attrs2        (:attrs rel2)
        common-attrs  (vec (intersect-keys (:attrs rel1) (:attrs rel2)))
        keep-attrs1   (keys attrs1)
        keep-attrs2   (->> attrs2
                        (reduce-kv (fn keeper [vec k _]
                                     (if (attrs1 k)
                                       vec
                                       (conj! vec k)))
                          (transient []))
                        persistent!) ; keys in attrs2-attrs1
        keep-idxs1    (to-array (vals attrs1))
        keep-idxs2    (to-array (->Eduction (map attrs2) keep-attrs2)) ; vals in attrs2-attrs1 by keys
        key-fn1       (tuple-key-fn attrs1 common-attrs)
        key-fn2       (tuple-key-fn attrs2 common-attrs)
        hash          (hash-attrs key-fn1 tuples1)
        new-tuples    (->>
                        tuples2
                        (reduce (fn outer [acc tuple2]
                                  (let [key (key-fn2 tuple2)]
                                    (if-some [tuples1 #?(:clj (hash key) :cljs (get hash key))]
                                      (reduce (fn inner [acc tuple1]
                                                (conj! acc (join-tuples tuple1 keep-idxs1 tuple2 keep-idxs2)))
                                        acc tuples1)
                                      acc)))
                          (transient []))
                        (persistent!))]
    (Relation. (zipmap (concat keep-attrs1 keep-attrs2) (range))
      new-tuples)))
