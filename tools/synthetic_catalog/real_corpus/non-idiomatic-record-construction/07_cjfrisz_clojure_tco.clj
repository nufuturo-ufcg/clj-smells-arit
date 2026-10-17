(ns ctco.expr.app
  (:require [ctco.expr
             cont simple thunk]
            [ctco.protocol :as proto]
            [ctco.util :as util])
  (:import [ctco.expr.cont
            Cont AppCont]
           [ctco.expr.simple
            Simple]
           [ctco.expr.thunk
            Thunk]))

(defrecord App [rator rand*]    
  proto/PCpsSrs
  (cps-srs [this k]
    (letfn [(cps-rand* [rand-in* rand-out*]
              (if (nil? (seq rand-in*))
                  (conj rand-out* k)
                  (let [fst (first rand-in*)
                        nxt (next rand-in*)]
                    (if (util/trivial? fst)
                        (recur nxt (conj rand-out* (proto/cps-triv fst)))
                        (let [s (util/new-var "s")]
                          (proto/cps-srs
                           fst
                           (Cont. s (cps-rand* nxt (conj rand-out* s)))))))))]
      (let [RAND* (cps-rand* (:rand* this) [])]
        (if (util/trivial? (:rator this))
            (App. (proto/cps-triv (:rator this)) RAND*)
            (proto/PCpsSrs (:rator this) (Cont. (util/new-var "s") RAND*))))))

  proto/PLoadTrampoline
  (load-tramp [this tramp]
    (proto/walk-expr this #(proto/load-tramp % tramp) nil))

  proto/PRecurify
  (recurify [this name arity tail?]
    (let [rator (:rator this)
          rand* (:rand* this)
          RAND* (mapv #(proto/recurify % nil nil false) rand*)]
      (if (and (= rator name) (= (count rand*) arity))
;; NB: seems like a manifest constant
;; NB: maybe fix this with the globals file
          (App. (Simple. 'recur) RAND*)
          (App. (proto/recurify rator nil nil false) RAND*))))
    
  proto/PThunkify
  (thunkify [this]
;; NB: seems like a manifest constant
;; NB: maybe fix this with the globals file
    (let [APP (proto/walk-expr this proto/thunkify nil)]
      (if (= (:rator this) (Simple. 'recur))
          APP
          (Thunk. APP))))

  proto/PUnparse
  (unparse [this]
    `(~(proto/unparse (:rator this)) ~@(map proto/unparse (:rand* this))))

  proto/PUnRecurify
  (unrecurify [this name]
;; NB: seems like a manifest constant
;; NB: maybe fix this with the globals file
    (let [recur-rec (Simple. 'recur)
          unrecurify #(proto/unrecurify % name)]
      (if (= (:rator this) recur-rec)
          (App. recur-rec (mapv unrecurify (:rand* this)))
          (proto/walk-expr this unrecurify nil))))

  proto/PWalkable
  (walk-expr [this f _]
    (App. (f (:rator this)) (mapv f (:rand* this)))))
