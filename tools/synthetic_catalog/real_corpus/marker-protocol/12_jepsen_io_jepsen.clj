(ns jepsen.antithesis
  "Provides support for running Jepsen tests in Antithesis. Provides an RNG,
  lifecycle hooks, and assertions.

  ## Randomness

  You should wrap your entire program in `(with-rng ...)`. This does nothing in
  ordinary environments, but in Antithesis, it replaces the jepsen.random RNG
  with one powered by Antithesis.

  ## Lifecycle

  Call `setup-complete!` once the test is ready to begin--for instance, at the
  end of Client/setup. Call `event!` to signal interesting things have
  happened.

  ## Assertions

  Assertions begin with `assert-`, and take an expression, a message, and data
  to include if the assertion fails. For instance:

    (assert-always! (not (db-corrupted?)) \"DB corrupted\" {:db \"foo\"})

  ## Wrappers

  Wrap your test map in `(a/test test-map)`. This wraps both the client and
  checkers with Antithesis instrumentation. You can also do this selectively
  using `checker` and `client`."
  (:refer-clojure :exclude [test])
  (:require [clojure [pprint :refer [pprint]]
                     [string :as str]]
            [clojure.java.io :as io]
            [clojure.tools.logging :refer [info]]
            [jepsen [checker :as checker]
                    [client :as client]
                    [db :as db]
                    [generator :as gen]
                    [os :as os]
                    [random :as rand]])
  (:import (com.antithesis.sdk Assert
                               Lifecycle)
           (com.fasterxml.jackson.databind ObjectMapper)
           (com.fasterxml.jackson.databind.node ArrayNode
                                                ObjectNode)
           (java.io Writer)
           (java.util.random RandomGenerator
                             RandomGenerator$SplittableGenerator)
           (jepsen.antithesis Random)))

(defn antithesis?
  "Are we running in an Antithesis environment?"
  []
  (boolean (dir)))

(def choice-cardinality
  "When selecting long values, we consider something an Antithesis \"choice\"
  if it asks for at most this many elements."
  16)

(defn setup-complete!
  "Logs that we've started up. Only emits once per JVM run. Optionally takes a
  JSON-coerceable structure with details."
  ([]
   (setup-complete! nil))
  ([details]
   (Lifecycle/setupComplete (->json-node details))))

(defmacro assert-always
  "Asserts that expr is true every time, and that it's called at least once.
  Takes a map of data which is serialized to JSON."
  [expr message data]
  `(Assert/always (boolean ~expr)
                  ~message
                  (->json-node ~data)))

(defmacro assert-sometimes
  "Asserts that expr is true at least once, and that this assertion is reached."
  [expr message data]
  `(Assert/sometimes (boolean ~expr)
                     ~message
                     (->json-node ~data)))

(defmacro assert-reachable
  "Asserts that this line of code is reached at least once."
  [message data]
  `(Assert/reachable ~message (->json-node ~data)))

(defrecord Client [client]
  client/Client
  (open! [this test node]
    (Client. (client/open! client test node)))

  (setup! [this test]
    (let [r (client/setup! client test)]
      (setup-complete!)
      r))

  (invoke! [this test op]
    ; Every Jepsen run should do at *least* one invocation. We emit one of
    ; these for each kind of :f the client does. TODO: I'm not sure if we
    ; should provide details or not.
    ;
    ; TODO: is it acceptable for us to runtime generate the name for this
    ; assertion? My guess is that whatever magic instrumentation Antithesis'
    ; SDK is doing might expect compile-time constants here, but the SDK docs
    ; don't say.
    (assert-reachable (str "invoke "  (pr-str (:f op))) {})
    (let [op' (client/invoke! client test op)]
      ; We'd like at least one invocation to succeed.
      ;
      ; TODO: is this OK? It feels like Antithesis could trivially generate a
      ; counterexample by just making the system unavailable, and that's not
      ; actually a bug. What SHOULD I be doing here?
      (assert-sometimes (identical? :ok (:type op'))
                        (str "ok " (pr-str (:f op)))
                        ; We'll at least log the type it came back with?
                        (select-keys op' [:type]))
      op'))

  (teardown! [this test]
    (client/teardown! client test))

  (close! [this test]
    (client/close! client test)))

(defn client
  "Wraps a Jepsen Client in one which performs some simple Antithesis
  instrumentation. Calls `setup-complete!` after the client's setup is done,
  and issues a pair of assertions for every operation--once on invocation, and
  once on completion."
  [client]
  (if (antithesis?)
    (Client. client)
    client))

(defprotocol Checker
  "This protocol marks checker types which perform their own Antithesis
  assertions. The `checker+` wrapper skips over any checkers which satisfy this
  protocol, as well as their children.")

(extend-protocol Checker
  ; The Stats checker will fail on histories with no :ok operations of some
  ; type, which Antithesis will sometimes generate.
  jepsen.checker.Stats)

(defrecord AlwaysChecker [^String name checker]
  Checker
  checker/Checker
  (check [this test history opts]
    (let [r (checker/check checker test history opts)]
      (assert-always (true? (:valid? r)) name r)
      r)))

(defn checker
  "Wraps a Jepsen Checker in one which asserts the results of the underlying
  checker are always valid. Takes a string name, which defaults to \"checker\";
  this is used for the Antithesis assertion."
  ([checker-]
   (checker "checker" checker-))
  ([name checker]
   (if (antithesis?)
     (AlwaysChecker. name checker)
     checker)))

(defn checker+
  "Rewrites a Jepsen Checker, wrapping every Checker in its own Antithesis
  Checker, which asserts validity of that specific checker. Each checker gets a
  name derived from the path into that data structure it took to get there.

  Ignores any object which satisfies the Checker protocol. You can use this to
  flag checkers you don't want to add assertions for, or which implement their
  own assertions."
  ([c]
   (checker+ ["checker"] c))
  ([path c]
   ; (prn :rewrite path c)
   (let [; We often rewrite singleton maps; in these cases, don't bother
         ; propagating paths. For example, a Compose has only a single key
         ; :checkers, and we don't need to generate names like "checker
         ; :checkers :cat"; we can just do "checker :cat".
         branch? (and (coll? c) (< 1 (count c)))
         ; Rewrite child forms
         c' (cond ; Already aware of Antithesis checking; don't explore any
                  ; deeper
                  (satisfies? Checker c)
                  c

                  ; For maps and records, rewrite each value, using the key as
                  ; our path
                  (map? c)
                  (reduce (fn [c [k v]]
                            (assoc c k (checker+
                                         (if branch?
                                           (conj path (pr-str k))
                                           path)
                                         v)))
                          c
                          c)

                  ; For sets, rewrite without changing path
                  (set? c)
                  (reduce (fn [c v]
                            (conj c (checker+ path v)))
                          (empty c)
                          c)

                  ; For sequential collections, rewrite without changing path
                  (sequential? c)
                  (reduce (fn [c v]
                            (conj c (checker+ path v)))
                          (empty c)
                          c)

                  ; Everything else bottoms out in itself
                  true
                  c)]
     ; Now, consider the rewritten thing.
     (cond ; Our Checkers handle their own assertions.
           (satisfies? Checker c')
           c'

           ; If it's a Jepsen checker, wrap it, using the current path as our
           ; name.
           (satisfies? checker/Checker c')
           (checker (str/join " " path) c')

           ; Otherwise, return unchanged.
           true
           c'))))

(defrecord EarlyTerminationGen [^long interval
                                ^double probability
                                ^long remaining
                                gen]
  gen/Generator
  (op [this test ctx]
    (if (and (= 0 remaining)
             ; This might be a little awkward if EarlyTerm winds up nested
             ; inside a branching structure; different calls to op would return
             ; different results. Probably fine, but might be worth
             ; stabilizing.
             (rand/bool probability))
        ; Done
        nil
        ; Keep going
        (when-let [[op gen'] (gen/op gen test ctx)]
          (if (identical? :pending op)
            [:pending this]
            [op (EarlyTerminationGen. interval
                                      probability
                                      (mod (dec remaining) interval)
                                      gen')]))))

  (update [this test ctx event]
    (EarlyTerminationGen. interval probability remaining
                          (gen/update gen test ctx event))))

(defn early-termination-generator
  "Wraps a Jepsen generator in one which, every so often, asks Jepsen's
  random source whether it ought to terminate. This allows Antithesis to
  perform a sort of weak 'online checking', as opposed to always running for
  the full (e.g.) time limit. Options are a map of:

    :interval    How many operations to emit before flipping a coin
    :probability The probability of early termination at each interval. This is
  only useful for extremely low/high probabilities; everything in the moderate
  range (see choice-cardinality) winds up being a coin toss."
  [{:keys [interval probability]} gen]
  (EarlyTerminationGen. (long interval)
                        (double (or probability 0.5))
                        (long interval)
                        gen))

(defn test
  "Prepares a Jepsen test for running in Antithesis. When running inside
  Antithesis, this:

  1. Replaces the OS with a no-op
  2. Repaces the DB with a no-op
  3. Replaces the SSH system with a stub.

  You likely also want to wrap the client in `client`, parts of the checker in
  `checker`, and possibly the generator in `early-termination-generator`."
  [test]
  (if (antithesis?)
    (assoc test
           :os os/noop
           :db db/noop
           :ssh {:dummy? true})
    test))
