(ns immutability-violation)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Using def inside a function to redefine a global var
(def countries {})
(defn update-country [country]
  (def countries (assoc countries (:name country) country)))

;; Example 2: Using alter-var-root to mutate a var at runtime
(def counter 0)
(defn increment-counter []
  (alter-var-root #'counter inc))

;; Example 3: Resetting an atom inside a function without returning new state
(def state (atom {}))
(defn update-state [k v]
  (reset! state (assoc @state k v)))

;; Example 4: Using set! to mutate a local Java field (mutable state) (Analisar hidden-side-effect; OBS: Possivelmente esse caso deveria estar nos casos para java)
(defn mutate-java-field [^java.util.concurrent.atomic.AtomicInteger ai]
  (set! (.value ai) 10))

;; Example 5: Using intern to dynamically redefine a var inside a function(
(defn redefine-var []
  (intern *ns* 'my-var 42))

;; Example 6: Defining a var inside a let binding (redefining global var) (not-detected)
(let [x 1]
  (def x 2))

;; Example 7: Using defonce inside a function to redefine a var multiple times
(defonce config {:debug false})
(defn update-config []
  (defonce config {:debug true}))

;; Example 8: Using binding to dynamically rebind a var and causing global state change
(def ^:dynamic *log-level* :info)
(defn change-log-level []
  (binding [*log-level* :debug]
    (def ^:dynamic *log-level* *log-level*)))


;; Example 9: Mutating a Java array directly inside a function 
(defn mutate-array [arr idx val]
  (aset arr idx val))


;; Example 10: Rebinding global var inside recursive function (def inside recursion)
(defn recursive-redef [n]
  (if (zero? n)
    0
    (do
      (def counter (inc n))
      (recursive-redef (dec n)))))

;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 11: Pure function returning new map without mutating any var
(defn update-country-pure [countries country]
  (assoc countries (:name country) country))

;; Example 12: Proper use of atom with swap! in controlled manner (detected)
(def state (atom {}))
(defn safe-update-state [k v]
  (swap! state assoc k v))

;; Example 13: Using let to create local bindings without side effects
(defn compute-sum [nums]
  (let [s (reduce + nums)]
    s))

;; Example 14: Pure function using recursion without side effects
(defn factorial [n]
  (if (<= n 1)
    1
    (* n (factorial (dec n)))))


;; Example 15: Using agents with send safely, no side effects in action fn
(def my-agent (agent 0))
(defn proper-agent-update []
  (send my-agent inc))

;; Example 16: Using binding for thread-local dynamic var changes only
(def ^:dynamic *user* nil)
(defn with-user [user f]
  (binding [*user* user]
    (f)))

;; Example 17: Pure function manipulating Java immutable types (e.g., String)
(defn append-string [s suffix]
  (str s suffix))

;; Example 18: Macro expanding to pure, immutable code (no mutation)
(defmacro pure-macro [x]
  `(+ ~x 1))
(defn use-pure-macro [n]
  (pure-macro n))

;; Example 19: Using letfn to define local recursive functions without mutation
(defn fib [n]
  (letfn [(fib-inner [k a b]
            (if (zero? k)
              a
              (fib-inner (dec k) b (+ a b))))]
    (fib-inner n 0 1)))

;; Example 20: Passing state explicitly through function parameters
(defn update-map [m k v]
  (assoc m k v))

;; Example 21: Immutable update of nested maps using update-in
(defn update-nested [m ks f]
  (update-in m ks f))

