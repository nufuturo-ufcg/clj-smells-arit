(ns latte.search
  "A Search facility for LaTTe
  (searching theorems, definitions, etc.)
  "
  (:require [latte.utils :as u])
)

(defn variable? [v]
  (if-not (symbol? v)
    false
    (let [sname (name v)]
      (and (> (count sname) 1)
           (= (first sname) \?)))))

(defn simple-variable? [v]
  (and (variable? v)
       (not (#{\* \+ \?} (last (name v))))))

(defn star-variable? [v]
  (and (variable? v)
       (= (last (name v)) \*)))

(defn ext-variable-name [sv]
  (let [svname (name sv)]
    (subs svname 1 (dec (count svname)))))

(defn ext-variable-sym [sv]
  (symbol (str "?" (ext-variable-name sv))))

(defn opt-variable? [v]
  (and (variable? v)
       (= (last (name v)) \?)))

(defn plus-variable? [v]
  (and (variable? v)
       (= (last (name v)) \+)))

(defn match 
  ([patt term] (match {} patt term))
  ([env patt term]
   (cond
     (simple-variable? patt) (match-var env patt term)
     (list? patt) (if-not (list? term)
                    false
                    (match-list env patt term))
     :else
     (if (= patt term)
       env
       false))))

(defn match-var [env var term]
  (if-let [prev-term (get env var)]
    (if (= prev-term term)
      env
      false)
    ;; no previous binding
    (if (= var '?_)
      env
      (assoc env var term))))

(defn match-list [env patts terms]
  (if (seq patts)
    (if (seq terms)
      (let [min-count (count-min-expected-terms patts)]
        (if (< (count terms) min-count)
          false ; not enough terms in list
          (let [patt (first patts)
                term (first terms)]
            (cond
              (opt-variable? patt)
              (if-let [env' (and (> (count terms) min-count)
                                 (match env (ext-variable-sym patt) term))]
                (recur env' (rest patts) (rest terms))
                (recur env (rest patts) terms))

              (star-variable? patt)
              (if (= (count terms) min-count)
                (recur env (rest patts) terms)
                (let [ext-var (ext-variable-sym patt)]
                  (do (when (get env ext-var)
                        (throw (ex-info "Multiple occurrence of zero-or-many variable" {:variable patt                                                                          :env env})))
                    (let [[matches terms'] (match-star ext-var env terms min-count)]
                      (recur (if (= ext-var '?_)
                               env
                               (assoc env ext-var matches))
                             (rest patts) 
                             terms')))))
              
              (plus-variable? patt)
              (let [ext-var (ext-variable-sym patt)]
                (do (when (get env ext-var)
                      (throw (ex-info "Multiple occurrence of one-or-many variable" {:variable patt                                                                          :env env})))             
                    (let [[matches terms'] (match-star ext-var env terms (dec min-count))]
                      (if (zero? (count matches))
                        false ; not enough matches                     
                        (recur (if (= ext-var '?_)
                                 env
                                 (assoc env ext-var matches))
                               (rest patts) terms')))))

              :else ; other kind of pattern
              (if-let [env' (match env patt term)]
                (recur env' (rest patts) (rest terms))
                false)))))
      ;; no element in term
      false)
    ;; nothing left to match
    (if (seq terms)
      false ; unmatched terms
      env)))

(defn match-star [sv env terms min-count]
  (loop [terms terms, nb-remain (- (count terms) min-count) matches []]
    (if (and (> nb-remain 0) (seq terms))              
      (if-let [env' (match env sv (first terms))]
        (recur (rest terms) (dec nb-remain) (conj matches (get env' sv)))
        [matches terms])
      [matches terms])))

(defn count-min-expected-terms [patts]
  (loop [patts patts, count 0]
    (if (seq patts)
      (if (or (star-variable? (first patts))
              (opt-variable? (first patts)))
        (recur (rest patts) count)
        ;; otherwise count is incremented
        (recur (rest patts) (inc count)))
      ;; no more patterns
      count)))
