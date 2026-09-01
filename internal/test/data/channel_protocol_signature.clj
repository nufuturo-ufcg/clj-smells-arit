(ns channel-protocol-signature)

(defprotocol Port
  (put! [port value callback]))
