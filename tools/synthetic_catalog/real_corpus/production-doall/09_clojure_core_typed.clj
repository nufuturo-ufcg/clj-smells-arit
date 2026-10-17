(ns ^:skip-wiki clojure.core.typed.checker.cs-gen
  (:require [clojure.core.typed.checker.utils :as u]
            [clojure.core.typed.contract-utils :as con]
            [clojure.core.typed.errors :as err]
            [clojure.core.typed.coerce-utils :as coerce]
            [clojure.core.typed.checker.type-rep :as r]
            [clojure.core.typed.checker.type-ctors :as c]
            [clojure.core.typed.checker.filter-rep :as fr]
            [clojure.core.typed.checker.filter-ops :as fo]
            [clojure.core.typed.checker.object-rep :as or]
            ; use subtype? utility defined in this namespace
            [clojure.core.typed.checker.jvm.subtype :as sub]
            [clojure.core.typed.checker.jvm.parse-unparse :as prs]
            [clojure.core.typed.checker.cs-rep :as cr]
            [clojure.core.typed.current-impl :as impl]
            [clojure.core.typed.util-vars :as vs]
            [clojure.core.typed.checker.dvar-env :as denv]
            [clojure.core.typed.checker.frees :as frees]
            [clojure.core.typed.checker.free-ops :as free-ops]
            [clojure.core.typed.checker.promote-demote :as prmt]
            [clojure.core.typed.checker.subst :as subst]
            [clojure.core.typed.checker.indirect-ops :as ind]
            [clojure.core.typed.checker.indirect-utils :as ind-u]
            [clojure.core.typed :as t :refer [letfn>]]
            [clojure.set :as set])
  (:import (clojure.core.typed.checker.type_rep F DataType Function Protocol Bounds FlowSet TCResult HSequential)
           (clojure.core.typed.checker.cs_rep c cset dcon dmap cset-entry)))

(defn ^:private gen-repeat [times repeated]
  (reduce (fn [acc cur]
            (concat acc cur))
          []
          (repeat times repeated)))

(defn ^:private subtype? [s t]
  (sub/subtype? s t))

(defn fail! [s t]
  (throw u/cs-gen-exn))

(defn meet [s t] (c/In s t))

(defn join [s t] (c/Un s t))

(defn c-meet 
  ([c1 c2] (c-meet c1 c2 nil))
  ([{S  :S X  :X T  :T bnds  :bnds :as c1}
    {S* :S X* :X T* :T bnds* :bnds :as c2}
    var]
   (when-not (or var (= X X*))
     (err/int-error (str "Non-matching vars in c-meet:" X X*)))
   (when-not (= bnds bnds*)
     (err/int-error (str "Non-matching bounds in c-meet:" bnds bnds*)))
   (let [;_ (prn "joining" S S* (mapv r/infer-any? [S S*]))
         S (join S S*)
         ;_ (prn "joined:" S)
         ;_ (prn "meeting" T T* (mapv r/infer-any? [T T*]))
         T (meet T T*)
         ;_ (prn "meet:" T)
         ]
     (when-not (subtype? S T)
       (fail! S T))
     (cr/c-maker S (or var X) T bnds))))

(defn dcon-meet [dc1 dc2]
  {:pre [(cr/dcon-c? dc1)
         (cr/dcon-c? dc2)]
   :post [(cr/dcon-c? %)]}
  (cond
    (and (cr/dcon-exact? dc1)
         (or (cr/dcon? dc2) 
             (cr/dcon-exact? dc2)))
    (let [{fixed1 :fixed rest1 :rest} dc1
          {fixed2 :fixed rest2 :rest} dc2]
      (when-not (and rest2 (= (count fixed1) (count fixed2)))
        (fail! fixed1 fixed2))
      (cr/dcon-exact-maker
        (doall
          (let [vector' (t/ann-form vector [c c -> '[c c]])]
            (t/for
              [[c1 c2] :- '[c c], (map vector' fixed1 fixed2)]
              :- c
              (c-meet c1 c2 (:X c1)))))
        (c-meet rest1 rest2 (:X rest1))))
    ;; redo in the other order to call the first case
    (and (cr/dcon? dc1)
         (cr/dcon-exact? dc2))
    (dcon-meet dc2 dc1)

    (and (cr/dcon? dc1)
         (not (:rest dc1))
         (cr/dcon? dc2)
         (not (:rest dc2)))
    (let [{fixed1 :fixed} dc1
          {fixed2 :fixed} dc2]
      (when-not (= (count fixed1) (count fixed2))
        (fail! fixed1 fixed2))
      (cr/dcon-maker
        (doall
          (for [[c1 c2] (map vector fixed1 fixed2)]
            (c-meet c1 c2 (:X c1))))
        nil))

    (and (cr/dcon? dc1)
         (not (:rest dc1))
         (cr/dcon? dc2))
    (let [{fixed1 :fixed} dc1
          {fixed2 :fixed rest :rest} dc2]
      (assert rest)
      (when-not (>= (count fixed1) (count fixed2))
        (fail! fixed1 fixed2))
      (cr/dcon-maker
        (let [vector' (t/inst vector c c t/Any t/Any t/Any t/Any)]
          (doall
            (t/for
              [[c1 c2] :- '[c c], (map vector' fixed1 (concat fixed2 (repeat rest)))]
              :- c
              (c-meet c1 c2 (:X c1)))))
        nil))

    (and (cr/dcon? dc1)
         (cr/dcon? dc2)
         (not (:rest dc2)))
    (dcon-meet dc2 dc1)

    (and (cr/dcon? dc1)
         (cr/dcon? dc2))
    (let [{fixed1 :fixed rest1 :rest} dc1
          {fixed2 :fixed rest2 :rest} dc2
          [shorter longer srest lrest]
          (if (< (count fixed1) (count fixed2))
            [fixed1 fixed2 rest1 rest2]
            [fixed2 fixed1 rest2 rest1])]
      (cr/dcon-maker
        (let [vector' (t/inst vector c c t/Any t/Any t/Any t/Any)]
          (doall
            (t/for
              [[c1 c2] :- '[c c], (map vector' longer (concat shorter (repeat srest)))]
              :- c
              (c-meet c1 c2 (:X c1)))))
        (c-meet lrest srest (:X lrest))))

    (and (cr/dcon-dotted? dc1)
         (cr/dcon-dotted? dc2))
    (let [{fixed1 :fixed c1 :dc {bound1 :name} :dbound} dc1
          {fixed2 :fixed c2 :dc {bound2 :name} :dbound} dc2]
      (when-not (and (= (count fixed1) (count fixed2))
                     (= bound1 bound2))
        (fail! bound1 bound2))
      (cr/dcon-dotted-maker 
        (let [vector' (t/inst vector c c t/Any t/Any t/Any t/Any)]
          (doall 
            (t/for
              [[c1 c2] :- '[c c], (map vector' fixed1 fixed2)]
              :- c
              (c-meet c1 c2 (:X c1)))))
        (c-meet c1 c2 bound1) bound1))

    (and (cr/dcon? dc1)
         (cr/dcon-dotted? dc2))
    (fail! dc1 dc2)

    (and (cr/dcon-dotted? dc1)
         (cr/dcon? dc2))
    (fail! dc1 dc2)

    (and (cr/dcon-repeat? dc1)
         (cr/dcon? dc2)
         (not (:rest dc2)))
    (let [{fixed1 :fixed repeated :repeat} dc1
          {fixed2 :fixed} dc2
          fixed1-count (count fixed1)
          fixed2-count (count fixed2)
          repeat-count (count repeated)
          diff (- fixed2-count fixed1-count)]
      (assert repeated)
      (when-not (and (>= fixed2-count fixed1-count)
                    (zero? (rem diff repeat-count)))
        (fail! fixed1 fixed2))
      (cr/dcon-repeat-maker
        (let [vector' (t/inst vector c c Any Any Any Any)]
          (doall
            (t/for 
              [[c1 c2] :- '[c c], (map vector'
                                       fixed2
                                       (concat fixed1
                                               (gen-repeat (quot diff repeat-count) repeated)))]
              :- c
              (c-meet c1 c2 (:X c1)))))
        repeated))
    (and (cr/dcon-repeat? dc2)
         (cr/dcon? dc1)
         (not (:rest dc1)))
    (dcon-meet dc2 dc1)

    (every? cr/dcon-repeat? [dc1 dc2])
    (let [[{short-fixed :fixed short-repeat :repeat}
           {long-fixed :fixed long-repeat :repeat}]
          (sort-by (fn [x] (-> x :fixed count)) [dc1 dc2])
          s-fixed-count (count short-fixed)
          l-fixed-count (count long-fixed)
          s-repeat-count (count short-repeat)
          l-repeat-count (count long-repeat)
          diff (- l-fixed-count s-fixed-count)
          _ (assert (= s-repeat-count l-repeat-count))
          vector' (t/inst vector c c Any Any Any Any)
          merged-repeat (t/for [[c1 c2] :- '[c c], (map vector' short-repeat long-repeat)]
                              :- c
                              (c-meet c1 c2 (:X c1)))]
      (assert (zero? (rem diff s-repeat-count)))
      (cr/dcon-repeat-maker
        (doall
          (t/for 
            [[c1 c2] :- '[c c], (map vector'
                                     long-fixed
                                     (concat short-fixed
                                             (gen-repeat (quot diff s-repeat-count) short-repeat)))]
            :- c
            (c-meet c1 c2 (:X c1))))
        merged-repeat))

    :else (err/nyi-error (str "NYI dcon-meet " dc1 dc2))))
