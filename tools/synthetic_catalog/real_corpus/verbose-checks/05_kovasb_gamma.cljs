(ns gamma.ast)

(defn head [x] (:head x))

(defn body [x] (:body x))

(def term-counter (atom 0))

(defn gen-term-id [] (let [id (swap! term-counter inc)] {:tag :id :id id}))

(defn term? [x]
  (and (map? x) (= :term (:tag x))))

(defn literal [x]
  (let [type (cond
               (clojure.core/or (= true x) (= false x)) :bool
               ;(integer? x) :int
               (number? x) :float
               (map? x) (:type x)
               :default (throw (js/Error. (str "Invalid literal: " (pr-str x)))))]
    {:tag :term
     :head :literal
     :value x
     :type type
     :id (gen-term-id)}))

(defn term [h & args]
  {:tag :term
   :head h
   :body (map #(if (term? %) % (literal %)) args)
   :id (gen-term-id)})
