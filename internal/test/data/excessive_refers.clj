(ns excessive-refers)

;; ========== CASES THAT SHOULD BE DETECTED ==========

;; Example 1: Single namespace explicitly referring a massive list of core domain utility functions
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.bloated-utils-list
  (:require [com.my-app.utils.core :refer [format-date parse-int active? send-email log-info render-html compute-total]]))

;; Example 2: Fragmented pollution - importing generic functions from 6 different namespaces, cluttering local scope
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.successive-pollution
  (:require [com.my-app.database :refer [save]]
            [com.my-app.validator :refer [validate]]
            [com.my-app.cacher :refer [fetch]]
            [com.my-app.logger :refer [log]]
            [com.my-app.auditor :refer [audit]]
            [com.my-app.tracker :refer [track]]))

;; Example 3: Legacy :use directive combined with a specific but excessively large list of vars via :only
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.legacy-use-bloated
  (:use [clojure.set :only [union intersection difference subset? superset? project join]]))

;; Example 4: Stacking several references that hover right at individual limits, heavily inflating the local namespace
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.edge-case-stacking
  (:require [com.my-app.module-a :refer [a1 a2 a3 a4 a5]]
            [com.my-app.module-b :refer [b1 b2 b3 b4 b5]]
            [com.my-app.module-c :refer [c1 c2 c3 c4 c5]]))

;; Example 5: Explicitly pulling in a long sequence of environmental configuration bindings
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.config-overload
  (:require [com.my-app.env :refer [db-host db-port db-user db-pass api-key cache-ttl worker-count]]))

;; Example 6: Bloated mathematical operations list (when not matching analytics or DSL exceptions)
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.service.billing
  (:require [com.my-app.utils.math :refer [add subtract multiply divide round-up apply-discount calculate-interest]]))

;; Example 7: Pulling excessive user-management operations instead of using a clean alias prefix
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.handler.user
  (:require [com.my-app.domain.users :refer [create-user update-user delete-user disable-user activate-user get-profile]]))

;; Example 8: Successive refers from sub-modules within the same bounded context, leading to keyword collisions
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.checkout.flow
  (:require [com.my-app.checkout.cart :refer [add-item remove-item empty-cart]]
            [com.my-app.checkout.pricing :refer [get-price apply-coupon]]
            [com.my-app.checkout.shipping :refer [calc-shipping select-carrier]]))

;; Example 9: Explicitly referring too many mock factories in a service implementation file
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.services.mocked
  (:require [com.my-app.mocks :refer [mock-user mock-order mock-payment mock-cart mock-address mock-invoice]]))

;; Example 10: Bloated list of formatting actions that bypasses simple code-splitting rules
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.view.formatter
  (:require [com.my-app.transformers :refer [date->str str->int int->money money->str float->percent currency->symbol]]))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========

;; Example 1: Idiomatic usage using an explicit namespace alias with zero refers
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.clean-alias
  (:require [clojure.string :as str]))

;; Example 2: Safe usage referencing only a single, highly specific function from a module
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.minimal-refer
  (:require [com.my-app.utilities :refer [calculate-tax]]))

;; Example 3: Multiple explicit requirements using strict namespace aliasing to avoid collisions
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.multi-alias
  (:require [com.my-app.users :as users]
            [com.my-app.orders :as orders]
            [com.my-app.payments :as payments]))

;; Example 4: Referencing a small, targeted pair of complementary extraction functions
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.targeted-pair
  (:require [clojure.edn :refer [read-string write-string]]))

;; Example 5: Requiring a namespace with an alias and using :refer strictly for a single conditional macro
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.macro-refer
  (:require [com.my-app.async-helpers :as async :refer [with-timeout]]))

;; Example 6: Requiring multiple namespaces but maintaining a very low number of refers (well under the threshold) for each
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.well-behaved-refers
  (:require [com.my-app.auth :refer [current-user]]
            [com.my-app.roles :refer [admin?]]))

;; Example 7: Using the legacy :use directive safely by strictly limiting it to a single variable via :only
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.safe-use
  (:use [clojure.data :only [diff]]))

;; Example 8: Requiring standard Clojure namespaces using clean, native aliases without global scope extraction
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.standard-libs
  (:require [clojure.set :as set]
            [clojure.walk :as walk]
            [clojure.zip :as zip]))

;; Example 9: Mixing standard aliased requires with a safe, single-symbol utility verification
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.mixed-safe
  (:require [com.my-app.formatter :as fmt]
            [com.my-app.validator :refer [valid-uuid?]]))

;; Example 10: Declaring a clean namespace that loads zero external packages or symbols, relying purely on clojure.core
#_{:clj-kondo/ignore [:namespace-name-mismatch]}
(ns com.my-app.pure-core)