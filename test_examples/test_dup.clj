(ns test.dup)

(defn func1 [x]
  (let [a (+ x 1)
        b (* a 2)]
    (+ a b)))

(defn func2 [y]
  (let [a (+ y 1)
        b (* a 2)]
    (+ a b)))

(defn func3 [z]
  (let [c (+ z 1)
        d (* c 2)]
    (+ c d))) 