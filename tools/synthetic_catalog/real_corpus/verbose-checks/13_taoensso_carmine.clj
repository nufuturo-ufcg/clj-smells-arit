(ns ^:no-doc taoensso.carmine-v4.resp.read
  "Private ns, implementation detail."
  (:require
   [taoensso.encore :as enc]
   [taoensso.truss  :as truss]
   [taoensso.nippy  :as nippy]
   [taoensso.carmine-v4.resp.common :as com
    :refer [xs->in+ throw!]])

  (:import
   [java.io DataInputStream]
   [taoensso.carmine_v4.resp.common ReadOpts #_ReadThawed Parser]))

(let [sentinel-end-of-aggregate-stream com/sentinel-end-of-aggregate-stream]
  (defn- read-aggregate-by-ones
    [to ^ReadOpts read-opts read-reply ^DataInputStream in]
    (let [size-str (.readLine in)
          inner-read-opts (com/in-aggregate-read-opts read-opts)
          skip? (identical? (.-read-mode read-opts) :skip)]

      (if-let [stream? (= size-str "?")]

        ;; Streaming
        (enc/cond
          skip?
          (loop []
            (let [x (read-reply inner-read-opts in)]
              (if (identical? x sentinel-end-of-aggregate-stream)
                com/sentinel-skipped-reply
                (recur))))

          ;; Reducing parser
          :if-let [^Parser p (com/when-rf-parser (.-parser read-opts))]
          (let [rf ((.rfc p))
                init-acc (rf)]
            (loop [acc init-acc]
              (let [x (read-reply inner-read-opts in)]
                (if (identical? x sentinel-end-of-aggregate-stream)
                  (do    (rf acc)) ; Complete acc
                  (recur (rf acc x))))))

          :default
          (loop [acc (transient (empty to))]
            (let [x (read-reply inner-read-opts in)]
              (if (identical? x sentinel-end-of-aggregate-stream)
                (persistent!  acc)
                (recur (conj! acc x))))))

        ;; Not streaming
        (let [n (Integer/parseInt size-str)]
          (enc/cond
            (< n 0) com/sentinel-null-reply
            skip? (enc/reduce-n (fn [_ _] (read-reply inner-read-opts in)) 0 n)

            ;; Reducing parser
            :if-let [^Parser p (com/when-rf-parser (.-parser read-opts))]
            (let [rf ((.-rfc p))
                  init-acc (rf)]
              (rf ; Complete acc
                (enc/reduce-n
                  (fn [acc _n]
                    (rf acc (read-reply inner-read-opts in)))
                  init-acc
                  n)))

            (== n 0) to
            :default
            (enc/repeatedly-into to n
              #(read-reply inner-read-opts in))))))))
