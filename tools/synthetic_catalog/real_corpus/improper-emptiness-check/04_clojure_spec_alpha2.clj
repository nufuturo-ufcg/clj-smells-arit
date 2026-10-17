(ns ^{:skip-wiki true}
  clojure.alpha.spec.impl
  (:require
    [clojure.alpha.spec :as s]
    [clojure.alpha.spec.protocols :as protocols
     :refer [Spec conform* unform* explain* gen* with-gen* describe*
             Schema keyspecs*
             Select]]
    [clojure.alpha.spec.gen :as gen]
    [clojure.set :as set]
    [clojure.walk :as walk]))

(defn- resolve-form
  [form]
  (if (keyword? form)
    form
    (-> form s/resolve-spec s/form)))

(defn- resolve-forms
  [forms]
  (map resolve-form forms))

(defn- maybe-spec
  "spec-or-k must be a spec, regex or resolvable kw/sym, else returns nil."
  [spec-or-k]
  (let [s (or (and (ident? spec-or-k) (#'s/reg-resolve spec-or-k))
                (s/spec? spec-or-k)
                (s/regex? spec-or-k)
                nil)]
    (if (s/regex? s)
      (#'s/with-name s (#'s/spec-name s))
      s)))

(defn- the-spec
  "spec-or-k must be a spec, regex or kw/sym, else returns nil. Throws if unresolvable kw/sym"
  [spec-or-k]
  (or (maybe-spec spec-or-k)
      (when (ident? spec-or-k)
        (throw (Exception. (str "Unable to resolve spec: " spec-or-k))))))

(defn- recur-limit? [rmap id path k]
  (and (> (get rmap id) (::s/recursion-limit rmap))
         (contains? (set path) k)))

(defn- inck [m k]
  (assoc m k (inc (or (get m k) 0))))

(defn- dt
  ([pred x form] (dt pred x form nil))
  ([pred x form cpred?]
   (if pred
     (if-let [spec (the-spec pred)]
       (s/conform spec x)
       (if (ifn? pred)
         (if cpred?
           (pred x)
           (if (pred x) x ::s/invalid))
         (throw (Exception. (str (pr-str form) " is not a fn, expected predicate fn")))))
     x)))

(defn- explain-1 [form pred path via in v settings-key settings]
  ;;(prn {:form form :pred pred :path path :in in :v v})
  (let [pred (maybe-spec pred)]
    (if (s/spec? pred)
      (explain* pred path (if-let [name (#'s/spec-name pred)] (conj via name) via) in v settings-key settings)
      [{:path path :pred form :val v :via via :in in}])))

(defn- explain-pred-list
  [forms preds path via in x settings-key settings]
  (loop [ret x
         [form & forms] forms
         [pred & preds] preds]
    (when pred
      (let [nret (dt pred ret form)]
        (if (s/invalid? nret)
          (explain-1 form pred path via in ret settings-key settings)
          (recur nret forms preds))))))

(defn- tagged-ret [tag ret]
  (clojure.lang.MapEntry. tag ret))

(defn- as-regex-spec
  ([re-obj]
    (as-regex-spec re-obj nil))
  ([re-obj gfn]
   (with-meta
     re-obj
     {`protocols/conform* (fn [re x settings-key settings]
                            (if (or (nil? x) (sequential? x))
                              (let [ret (re-conform re (seq x))]
                                (if (and (vector? x) (map? ret))
                                  (with-meta ret {::s/into []})
                                  ret))
                              ::s/invalid))
      `protocols/unform* (fn [re x]
                           (let [ret (op-unform re x)]
                             (if (= (-> x meta ::s/into) [])
                               (vec ret)
                               ret)))
      `protocols/explain* (fn [re path via in x settings-key settings]
                            (if (or (nil? x) (sequential? x))
                              (re-explain path via in re (seq x) settings-key settings)
                              [{:path path :pred (#'s/res `#(or (nil? %) (sequential? %))) :val x :via via :in in}]))
      `protocols/gen* (fn [re overrides path rmap]
                        (if gfn
                          (gfn)
                          (re-gen re overrides path rmap (op-describe re))))
      `protocols/with-gen* (fn [re gfn]
                             (as-regex-spec re gfn))
      `protocols/describe* (fn [re]
                             (op-describe re))})))

(defn- accept [x] (as-regex-spec {::s/op ::s/accept :ret x}))

(defn- accept? [{:keys [::s/op]}]
  (= ::s/accept op))

(defn- pcat* [{[p1 & pr :as ps] :ps,  [k1 & kr :as ks] :ks, [f1 & fr :as forms] :forms, ret :ret, rep+ :rep+}]
  (when (every? identity ps)
    (if (accept? p1)
      (let [rp (:ret p1)
            ret (conj ret (if ks {k1 rp} rp))]
        (if pr
          (pcat* {:ps pr :ks kr :forms fr :ret ret})
          (accept ret)))
      (as-regex-spec {::s/op ::s/pcat, :ps ps, :ret ret, :ks ks, :forms forms :rep+ rep+}))))

(defn- pcat [& ps] (pcat* {:ps ps :ret []}))

(defn- rep* [p1 p2 ret splice form]
  (when p1
    (let [r (as-regex-spec {::s/op ::s/rep, :p2 p2, :splice splice, :forms form :id (java.util.UUID/randomUUID)})]
      (if (accept? p1)
        (assoc r :p1 p2 :ret (conj ret (:ret p1)))
        (assoc r :p1 p1, :ret ret)))))

(defn- amp-impl
  "Do not call this directly, use '&'"
  [re re-form preds pred-forms]
  (as-regex-spec {::s/op ::s/amp :p1 re :amp re-form :ps preds :forms pred-forms}))

(defn- filter-alt [ps ks forms f]
  (if (or ks forms)
    (let [pks (->> (map vector ps
                        (or (seq ks) (repeat nil))
                        (or (seq forms) (repeat nil)))
                   (filter #(-> % first f)))]
      [(seq (map first pks)) (when ks (seq (map second pks))) (when forms (seq (map #(nth % 2) pks)))])
    [(seq (filter f ps)) ks forms]))

(defn- alt* [ps ks forms]
  (let [[[p1 & pr :as ps] [k1 :as ks] forms] (filter-alt ps ks forms identity)]
    (when ps
      (let [ret (as-regex-spec {::s/op ::s/alt, :ps ps, :ks ks :forms forms})]
        (if (nil? pr)
          (if k1
            (if (accept? p1)
              (accept (tagged-ret k1 (:ret p1)))
              ret)
            p1)
          ret)))))

(defn- alts [& ps] (alt* ps nil nil))

(defn- alt2 [p1 p2] (if (and p1 p2) (alts p1 p2) (or p1 p2)))

(defn- noret? [p1 pret]
  (or (= pret ::s/nil)
        (and (#{::s/rep ::s/pcat} (::s/op (#'s/reg-resolve! p1))) ;;hrm, shouldn't know these
               (empty? pret))
        nil))

(defn- and-preds [x preds forms]
  (loop [ret x
         [pred & preds] preds
         [form & forms] forms]
    (if pred
      (let [nret (dt pred ret form)]
        (if (s/invalid? nret)
          ::s/invalid
          ;;propagate conformed values
          (recur nret preds forms)))
      ret)))

(defn- accept-nil? [p]
  (let [{:keys [::s/op ps p1 p2 forms] :as p} (#'s/reg-resolve! p)]
    (case op
      ::s/accept true
      nil nil
      ::s/amp (and (accept-nil? p1)
                   (let [ret (-> (preturn p1) (and-preds ps (next forms)))]
                     (not (s/invalid? ret))))
      ::s/rep (or (identical? p1 p2) (accept-nil? p1))
      ::s/pcat (every? accept-nil? ps)
      ::s/alt (some accept-nil? ps))))

(defn- preturn [p]
  (let [{[p0 & pr :as ps] :ps, [k :as ks] :ks, :keys [::s/op p1 ret forms] :as p} (#'s/reg-resolve! p)]
    (case op
      ::s/accept ret
      nil nil
      ::s/amp (let [pret (preturn p1)]
              (if (noret? p1 pret)
                ::s/nil
                (and-preds pret ps forms)))
      ::s/rep (add-ret p1 ret k)
      ::s/pcat (add-ret p0 ret k)
      ::s/alt (let [[[p0] [k0]] (filter-alt ps ks forms accept-nil?)
                  r (if (nil? p0) ::s/nil (preturn p0))]
              (if k0 (tagged-ret k0 r) r)))))

(defn- op-unform [p x]
  ;;(prn {:p p :x x})
  (let [{[p0 & pr :as ps] :ps, [k :as ks] :ks, :keys [::s/op p1 ret forms rep+ maybe] :as p} (#'s/reg-resolve! p)
        kps (zipmap ks ps)]
    (case op
      ::s/accept [ret]
      nil [(s/unform p x)]
      ::s/amp (let [px (reduce #(s/unform %2 %1) x (reverse ps))]
              (op-unform p1 px))
      ::s/rep (mapcat #(op-unform p1 %) x)
      ::s/pcat (if rep+
               (mapcat #(op-unform p0 %) x)
               (mapcat (fn [k]
                         (when (contains? x k)
                           (op-unform (kps k) (get x k))))
                       ks))
      ::s/alt (if maybe
              [(s/unform p0 x)]
              (let [[k v] x]
                (op-unform (kps k) v))))))

(defn- add-ret [p r k]
  (let [{:keys [::s/op ps splice] :as p} (#'s/reg-resolve! p)
        prop #(let [ret (preturn p)]
                (if (empty? ret) r ((if splice into conj) r (if k {k ret} ret))))]
    (case op
      nil r
      (::s/alt ::s/accept ::s/amp)
      (let [ret (preturn p)]
        ;;(prn {:ret ret})
        (if (= ret ::s/nil) r (conj r (if k {k ret} ret))))

      (::s/rep ::s/pcat) (prop))))

(defn- deriv
  [p x]
  (let [{[p0 & pr :as ps] :ps, [k0 & kr :as ks] :ks, :keys [::s/op p1 p2 ret splice forms amp] :as p} (#'s/reg-resolve! p)]
    (when p
      (case op
        ::s/accept nil
        nil (let [ret (dt p x p)]
              (when-not (s/invalid? ret) (accept ret)))
        ::s/amp (when-let [p1 (deriv p1 x)]
                (if (= ::s/accept (::s/op p1))
                  (let [ret (-> (preturn p1) (and-preds ps (next forms)))]
                    (when-not (s/invalid? ret)
                      (accept ret)))
                  (amp-impl p1 amp ps forms)))
        ::s/pcat (alt2 (pcat* {:ps (cons (deriv p0 x) pr), :ks ks, :forms forms, :ret ret})
                     (when (accept-nil? p0) (deriv (pcat* {:ps pr, :ks kr, :forms (next forms), :ret (add-ret p0 ret k0)}) x)))
        ::s/alt (alt* (map #(deriv % x) ps) ks forms)
        ::s/rep (alt2 (rep* (deriv p1 x) p2 ret splice forms)
                    (when (accept-nil? p1) (deriv (rep* p2 p2 (add-ret p1 ret nil) splice forms) x)))))))

(defn- op-describe [p]
  (let [{:keys [::s/op ps ks forms splice p1 rep+ maybe amp] :as p} (#'s/reg-resolve! p)]
    ;;(prn {:op op :ks ks :forms forms :p p})
    (when p
      (case op
        ::s/accept nil
        nil p
        ::s/amp (list* 'clojure.alpha.spec/& (resolve-form amp) (resolve-forms forms))
        ::s/pcat (if rep+
                 (list `s/+ rep+)
                 (cons `s/cat (mapcat vector (or (seq ks) (repeat :_)) (resolve-forms forms))))
        ::s/alt (if maybe
                (list `s/? maybe)
                (cons `s/alt (mapcat vector ks (resolve-forms forms))))
        ::s/rep (list (if splice `s/+ `s/*) (resolve-form forms))))))

(defn- op-explain [form p path via in input settings-key settings]
  ;;(prn {:form form :p p :path path :input input})
  (let [[x :as input] input
        {:keys [::s/op ps ks forms splice p1 p2] :as p} (#'s/reg-resolve! p)
        via (if-let [name (#'s/spec-name p)] (conj via name) via)
        insufficient (fn [path form]
                       [{:path path
                         :reason "Insufficient input"
                         :pred form
                         :val ()
                         :via via
                         :in in}])]
    (when p
      (case op
        ::s/accept nil
        nil (if (empty? input)
              (insufficient path form)
              (explain-1 form p path via in x settings-key settings))
        ::s/amp (if (empty? input)
                (if (accept-nil? p1)
                  (explain-pred-list forms ps path via in (preturn p1) settings-key settings)
                  (insufficient path (:amp p)))
                (if-let [p1 (deriv p1 x)]
                  (explain-pred-list forms ps path via in (preturn p1) settings-key settings)
                  (op-explain (:amp p) p1 path via in input settings-key settings)))
        ::s/pcat (let [pkfs (map vector
                               ps
                               (or (seq ks) (repeat nil))
                               (or (seq forms) (repeat nil)))
                     [pred k form] (if (= 1 (count pkfs))
                                     (first pkfs)
                                     (first (remove (fn [[p]] (accept-nil? p)) pkfs)))
                     path (if k (conj path k) path)
                     form (or form (op-describe pred))]
                 (if (and (empty? input) (not pred))
                   (insufficient path form)
                   (op-explain form pred path via in input settings-key settings)))
        ::s/alt (if (empty? input)
                (insufficient path (op-describe p))
                (apply concat
                       (map (fn [k form pred]
                              (op-explain (or form (op-describe pred))
                                          pred
                                          (if k (conj path k) path)
                                          via
                                          in
                                          input
                                          settings-key
                                          settings))
                            (or (seq ks) (repeat nil))
                            (or (seq forms) (repeat nil))
                            ps)))
        ::s/rep (op-explain (if (identical? p1 p2)
                            forms
                            (op-describe p1))
                          p1 path via in input
                          settings-key settings)))))

(defn- re-gen [p overrides path rmap f]
  ;;(prn {:op op :ks ks :forms forms})
  (let [origp p
        {:keys [::s/op ps ks p1 p2 forms splice ret id ::s/gfn] :as p} (#'s/reg-resolve! p)
        rmap (if id (inck rmap id) rmap)
        ggens (fn [ps ks forms]
                (let [gen (fn [p k f]
                            ;;(prn {:k k :path path :rmap rmap :op op :id id})
                            (when-not (and rmap id k (recur-limit? rmap id path k))
                              (if id
                                (gen/delay (re-gen p overrides (if k (conj path k) path) rmap (or f p)))
                                (re-gen p overrides (if k (conj path k) path) rmap (or f p)))))]
                  (map gen ps (or (seq ks) (repeat nil)) (or (seq forms) (repeat nil)))))]
    (or (when-let [gfn (or (get overrides (#'s/spec-name origp))
                               (get overrides (#'s/spec-name p) )
                               (get overrides path))]
            (case op
              (:accept nil) (gen/fmap vector (gfn))
              (gfn)))
          (when gfn
            (gfn))
          (when p
            (case op
              ::s/accept (if (= ret ::s/nil)
                         (gen/return [])
                         (gen/return [ret]))
              nil (when-let [g (#'s/gensub p overrides path rmap f)]
                    (gen/fmap vector g))
              ::s/amp (re-gen p1 overrides path rmap (op-describe p1))
              ::s/pcat (let [gens (ggens ps ks forms)]
                       (when (every? identity gens)
                         (apply gen/cat gens)))
              ::s/alt (let [gens (remove nil? (ggens ps ks forms))]
                      (when-not (empty? gens)
                        (gen/one-of gens)))
              ::s/rep (if (recur-limit? rmap id [id] id)
                      (gen/return [])
                      (when-let [g (re-gen p2 overrides path rmap forms)]
                        (gen/fmap #(apply concat %)
                                  (gen/vector g)))))))))

(defn- re-conform [p [x & xs :as data]]
  ;;(prn {:p p :x x :xs xs})
  (if (empty? data)
    (if (accept-nil? p)
      (let [ret (preturn p)]
        (if (= ret ::s/nil)
          nil
          ret))
      ::s/invalid)
    (if-let [dp (deriv p x)]
      (recur dp xs)
      ::s/invalid)))

(defn- re-explain [path via in re input settings-key settings]
  (loop [p re [x & xs :as data] input i 0]
    ;;(prn {:p p :x x :xs xs :re re}) (prn)
    (if (empty? data)
      (if (accept-nil? p)
        nil ;;success
        (op-explain (op-describe p) p path via in nil settings-key settings))
      (if-let [dp (deriv p x)]
        (recur dp xs (inc i))
        (if (accept? p)
          (if (= (::s/op p) ::s/pcat)
            (op-explain (op-describe p) p path via (conj in i) (seq data) settings-key settings)
            [{:path path
              :reason "Extra input"
              :pred (op-describe re)
              :val data
              :via via
              :in (conj in i)}])
          (or (op-explain (op-describe p) p path via (conj in i) (seq data) settings-key settings)
                [{:path path
                  :reason "Extra input"
                  :pred (op-describe p)
                  :val data
                  :via via
                  :in (conj in i)}]))))))
