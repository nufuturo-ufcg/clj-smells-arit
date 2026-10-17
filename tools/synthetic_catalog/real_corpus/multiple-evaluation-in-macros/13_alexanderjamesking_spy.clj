(ns ^{:author "Alexander James King",
      :doc "Assertion functions wrapping spy.core functions,
      providing assertions using clojure.test/is and messages
      to aid debugging when test expectations are not met.

      Macros are used to ensure the clojure.test/is macro
      appears in the calling code, otherwise line numbers
      reported on failure will show as failures in the
      library and not the calling code."}
  spy.assert
  (:require [spy.core :as spy]
            [spy.assert-messages :as assert-messages]
            [clojure.test :refer [is]]))

(defmacro called-with?
  [f & args]
  `(is (spy/called-with? ~f ~@args)
       (assert-messages/called-with ~f ~@args)))
