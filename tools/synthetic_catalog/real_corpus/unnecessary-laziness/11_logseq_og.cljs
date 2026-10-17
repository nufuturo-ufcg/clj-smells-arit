(ns logseq.graph-parser.block
  "Given mldoc ast, prepares block data in preparation for db transaction"
  (:require [clojure.set :as set]
            [clojure.string :as string]
            [clojure.walk :as walk]
            [datascript.core :as d]
            [logseq.graph-parser.config :as gp-config]
            [logseq.graph-parser.date-time-util :as date-time-util]
            [logseq.graph-parser.mldoc :as gp-mldoc]
            [logseq.graph-parser.property :as gp-property]
            [logseq.graph-parser.text :as text]
            [logseq.graph-parser.utf8 :as utf8]
            [logseq.graph-parser.util :as gp-util]
            [logseq.graph-parser.util.block-ref :as block-ref]
            [logseq.graph-parser.util.page-ref :as page-ref]))

(defn with-parent-and-left
  [page-id blocks]
  (let [[blocks other-blocks] (split-with
                               (fn [b]
                                 (not= "macro" (:block/type b)))
                               blocks)
        result (loop [blocks (map (fn [block] (assoc block :block/level-spaces (:block/level block))) blocks)
                      parents [{:page/id page-id     ; db id or a map {:block/name "xxx"}
                                :block/level 0
                                :block/level-spaces 0}]
                      result []]
                 (if (empty? blocks)
                   (map #(dissoc % :block/level-spaces) result)
                   (let [[block & others] blocks
                         level-spaces (:block/level-spaces block)
                         {:block/keys [uuid level parent] :as last-parent} (last parents)
                         parent-spaces (:block/level-spaces last-parent)
                         [blocks parents result]
                         (cond
                           (= level-spaces parent-spaces)        ; sibling
                           (let [block (assoc block
                                              :block/parent parent
                                              :block/left [:block/uuid uuid]
                                              :block/level level)
                                 parents' (conj (vec (butlast parents)) block)
                                 result' (conj result block)]
                             [others parents' result'])

                           (> level-spaces parent-spaces)         ; child
                           (let [parent (if uuid [:block/uuid uuid] (:page/id last-parent))
                                 block (cond->
                                         (assoc block
                                                :block/parent parent
                                                :block/left parent)
                                         ;; fix block levels with wrong order
                                         ;; For example:
                                         ;;   - a
                                         ;; - b
                                         ;; What if the input indentation is two spaces instead of 4 spaces
                                         (>= (- level-spaces parent-spaces) 1)
                                         (assoc :block/level (inc level)))
                                 parents' (conj parents block)
                                 result' (conj result block)]
                             [others parents' result'])

                           (< level-spaces parent-spaces)
                           (cond
                             (some #(= (:block/level-spaces %) (:block/level-spaces block)) parents) ; outdent
                             (let [parents' (vec (filter (fn [p] (<= (:block/level-spaces p) level-spaces)) parents))
                                   left (last parents')
                                   blocks (cons (assoc (first blocks)
                                                       :block/level (dec level)
                                                       :block/left [:block/uuid (:block/uuid left)])
                                                (rest blocks))]
                               [blocks parents' result])

                             :else
                             (let [[f r] (split-with (fn [p] (<= (:block/level-spaces p) level-spaces)) parents)
                                   left (first r)
                                   parent-id (if-let [block-id (:block/uuid (last f))]
                                               [:block/uuid block-id]
                                               page-id)
                                   block (cond->
                                           (assoc block
                                                  :block/parent parent-id
                                                  :block/left [:block/uuid (:block/uuid left)]
                                                  :block/level (:block/level left)
                                                  :block/level-spaces (:block/level-spaces left)))

                                   parents' (->> (concat f [block]) vec)
                                   result' (conj result block)]
                               [others parents' result'])))]
                     (recur blocks parents result))))]
    (concat result other-blocks)))
