(ns day8.re-frame-10x.panels.subs.subs
  (:require
   [clojure.string                                               :as string]
   [day8.re-frame-10x.inlined-deps.re-frame.v1v3v0.re-frame.core :as rf]
   [day8.re-frame-10x.navigation.epochs.subs                     :as epochs.subs]
   [day8.re-frame-10x.panels.traces.subs                         :as traces.subs]
   [day8.re-frame-10x.tools.metamorphic                          :as metam]))

(defn sub-type->value
  [sub-type]
  (case sub-type
    :sub/create  5
    :sub/run     4
    :sub/dispose 3
    :sub/not-run 2
    1))

(defn accumulate-sub-value
  "Calculate a sorting value for a series of subscription trace types."
  ;; A reader might reasonably ask, "Why are we going to all this work here?"
  ;; We calculate a custom value rather than just comparing two order vectors,
  ;; because the default compare logic for comparing vectors is to sort shorter
  ;; vectors above longer ones, whereas we want all CRR, CR, C orders to be
  ;; sorted adjacent to each other, in that order.
  ;;
  ;; The first sub type in the order is worth (n * 10^3),
  ;; then the next one (if it exists), is worth (n * 10^2), and so-on.
  [order]
  (loop [exp   3
         total 0
         order order]
    (if-let [sub-type (first order)]
      (recur (dec exp)
             (+ total (* (sub-type->value sub-type) (js/Math.pow 10 exp)))
             (rest order))
      total)))

(def accumulate-sub-value-memoized
  (memoize accumulate-sub-value))

(defn sub-sort-val [order-x order-y]
  ;; Note x and y are reversed here so that the "highest" sub orders get sorted first.
  (compare (accumulate-sub-value-memoized order-y)
           (accumulate-sub-value-memoized order-x)))

(defn prepare-pod-info
  "Returns sub info prepared for rendering in pods"
  [[sub-info sub-state] [subscription]]
  (let [remove-fn (if (= subscription ::intra-epoch-subs)
                    (fn [me] (nil? (:order (val me))))
                    (constantly false))
        subx      (->> sub-state
                       (remove remove-fn)
                       (map (fn [me] (let [state        (val me)
                                           subscription (:subscription state)
                                           sub          {:id         (key me)
                                                         :reagent-id (key me)
                                                         :layer      (get-in sub-info [(first subscription) :layer])
                                                         :path-data  subscription
                                                         :path       (pr-str subscription)
                                                         :order      (or (:order state) [:sub/not-run])
                                                         :sub/traits (:sub/traits state)}
                                           sub          (if (contains? state :value)
                                                          (assoc sub :value (:value state))
                                                          sub)
                                           sub          (if (contains? state :previous-value)
                                                          (assoc sub :previous-value (:previous-value state))
                                                          sub)]
                                       sub)))
                       (sort-by :order sub-sort-val)        ;; Also sort by subscription-id
                       #_(sort-by :path))]
    subx))
