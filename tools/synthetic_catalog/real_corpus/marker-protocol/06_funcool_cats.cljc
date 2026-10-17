(ns cats.protocols
  "A collection of protocols upon which the cats abstractions are built.

  NOTE: Functions of this namespace are not intended to be used directly.
  It is considered internal api.")

(defprotocol Context
  "A marker protocol for identifying the valid context types.")
