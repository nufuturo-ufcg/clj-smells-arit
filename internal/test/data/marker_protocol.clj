(ns marker-protocol)

(defprotocol EmptyMarker)

(defprotocol DocumentedMarker
  "Only documentation, no methods.")

(defprotocol RealProtocol
  (load-value [this]))

(defprotocol ExportedProtocol
  (^:export save-value [this value]))

(defn defprotocol [& _] :shadowed)
(defprotocol NotAProtocol)
