(ns overtone.repl.graphviz
  (:use [overtone.repl.debug :only [unified-sdef]]
        [overtone.sc.machinery.ugen.metadata.unaryopugen]
        [overtone.sc.machinery.ugen.metadata.binaryopugen]
        [overtone.helpers.system :only [mac-os? linux-os?]]
        [overtone.helpers.file :only [mk-tmp-dir! mk-path]]
        [clojure.java.shell]))

(defn- safe-name
  [n]
  (let [n (name n)]
    (.replaceAll (.toLowerCase (str n)) "[-]" "____")))

(defn- print-connection
  [n input ug]
  (cond
   (vector? input)      (dorun (map (fn [idx i]
                                      (when (associative? i)
                                        (println (str (:id i) ":__UG_NAME__ -> " (:id ug) ":" (safe-name n) "___" (safe-name (:name i)) "___" idx " ;"))))
                                    (range)
                                    input))
   (associative? input) (println (str (:id input) ":__UG_NAME__ -> " (:id ug) ":" (safe-name n) " ;"))))

(defn- generate-node-connections
  [ugs]
  (with-out-str
    (doseq [ug ugs]
      (doseq [[n input] (:inputs ug)]
        (print-connection n input ug)))))
