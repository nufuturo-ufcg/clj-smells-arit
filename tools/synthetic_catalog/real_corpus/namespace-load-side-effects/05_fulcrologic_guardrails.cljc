(ns com.fulcrologic.guardrails.config
  "Manages the guardrails configuration. Most of this is automatic, and is controlled by files you place on the
   filesystem and possibly reference by system properties (which you can set on the JVM with the -D option).

   Controlling Guardrails Checks at Runtime

   Normally if guardrails is enabled, then the macros (e.g. `>defn`) emit code that will cause checking to happen
   on every call. This is fine when you are working on files you own, but as your project grows, and you possibly
   bring in libraries that also use Guardrails you may find that runtime checking adds significant (and unacceptable)
   overhead to your development environment.

   As a library author, you can include a file named `guardrails-export.edn` in the root of your main source code
   (making sure it gets copied into your distribution jar at the top-level). This file can be used to auto-exclude
   namespaces or functions from runtime checks, so that users of your library do not take an unnecessary performance hit.

   The idea is that all internal implementation can have GR turned off because you tested your library
   and are sure you're doing internal cross-calls correctly.

   So, your library can leave checking turned on at the core API interface level.

   Say you have these namespaces in your library:

   ```
   com.project.impl
   com.project.other
   com.project.core
   ```

   You might include the following `guardrails-export.edn` in your `src` folder:

   ```
   {:exclude #{com.project.impl com.project.other}}
   ```

   Now users of your library that also use GR will not take a performance hit on your internal implementation.

   An as end user you can *always* choose to turn checking on/off anywhere/everywhere. See:

   allow-checks! - Enable checking on something that was excluded
   exclude-checks! - Disable checking on something
   clear-exclusions! - Enable checking EVERYWHERE
   reset-exclusions! - Bring the exclusions back to load-time default (library export values)
   excluded? - Check if a function/ns is currently being excluded

   Disabling Exclusions Globally (for CI/Testing)

   There are times when you might want the exclusions to not apply. For example as a library author running
   local tests from the CLI (e.g. kaocha) you don't want your exclusions being included.

   You can suppress startup messages with JVM `-Dguardrails.silent=true`.

   You can globally disable exclusions by choosing an alternate config file (with JVM `-Dguardrails.config=filename`)
   and adding a flag:

   ```
   {:throw?              true
    :disable-exclusions? true}
   ```

   This will cause a runtime check on all instrumented functions everywhere, including other libraries.

   Your alternative is to make sure that something like `clear-exclusions!` runs before any of your tests.

   Improving Performance with Throttled Checking

   Guardrails normally runs checks on every call. For functions that are called in tight loops this is too much, and
   leads to serious slowdowns. Excluding the fn or namespace fixes this, but then you have to visibility for problems.
   An alternative is to ask Guardrails to limit the number of checks to a specific rate/s. This can be done globally,
   by namespace, or by function (anywhere you can put guardrails config. The option is :guardrails/mcps (max
   checks per second). The first call is always checked, and then the rate is enforced at ns accuracy at each additional
   call. Of course this is affected by various factors (timer jitter, etc.), and will give an approximate behavior.

   The overhead of the throttling code is designed to be very light. It isn't present at all unless enabled, and
   adds somewhere around 30ns of overhead per call. Much less than the roughly 10 microsecond overhead of an actual
   spec/return value check (almost 1000x faster).

   ```
   (>defn f
     {:guardrails/mcps 100}
     [n]
     [int? => int?]
     ...)
   ```
   "
  #?(:cljs (:require-macros com.fulcrologic.guardrails.config))
  (:require
    [com.fulcrologic.guardrails.utils :as utils]
    #?@(:clj  [[clojure.edn :as edn]
               [clojure.set :as set]]
        :cljs [[cljs.env :as cljs-env]])))

#?(:bb  (require '[com.fulcrologic.guardrails.stubs.cljs-env :as cljs-env])
   :clj (try
          (ns-unalias (find-ns 'com.fulcrologic.guardrails.utils) 'cljs-env)
          (require '[cljs.env :as cljs-env])
          (catch Exception _
            (require '[com.fulcrologic.guardrails.stubs.cljs-env :as cljs-env]))))
