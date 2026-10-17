(ns immutability-violation-array-alias)
;; An alias of a defn parameter is the caller's array; a let that shadows the
;; parameter with a local array is not, and neither is an anonymous fn local.
(defn shadowed [buf]
  (let [buf (byte-array 4)
        ^bytes b buf]
    (aset b 0 (byte 1))
    b))
(defn param-alias [^bytes buf]
  (let [^bytes b buf]
    (aset b 0 (byte 1))))
(defn inner-fn-param [xs]
  (map (fn [buf] (let [^bytes b (byte-array 2)] (let [^bytes c b] (aset c 0 (byte 1)) c))) xs))
