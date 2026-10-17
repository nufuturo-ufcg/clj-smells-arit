(ns cortex.verify.nn.layers
  "Verify that layers do actually produce their advertised results."
  (:require
    [clojure.test :refer :all]
    [clojure.core.matrix :as m]
    [think.resource.core :as resource]
    [cortex.graph :as graph]
    [cortex.gaussian :as cu]
    [cortex.nn.layers :as layers]
    [cortex.nn.network :as network]
    [cortex.nn.traverse :as traverse]
    [cortex.nn.execute :as execute]
    [cortex.nn.compute-binding :as compute-binding]
    [cortex.verify.utils :as utils]))

(defn bind-test-network
  [description context batch-size & {:keys [bind-opts]
                                     :or {bind-opts {}}}]
  (let [network (network/linear-network description)]
    (-> network
        (compute-binding/bind-context-to-network
         (execute/current-backend)
         batch-size
         (traverse/training-traversal network :keep-non-trainable? true)
         (assoc bind-opts
                :disable-pooling? true)))))

(defn unpack-network
  [network test-layer-id]
  (let [traversal (compute-binding/traversal network)
        test-node (get-in network [:compute-graph :nodes test-layer-id])
        parameter-descriptions (->> (graph/get-node-arguments test-node)
                                    (filter #(= :parameter (get % :type))))
        parameters (->> (map (fn [{:keys [key] :as description}]
                               (let [parameter (get test-node key)
                                     buffer (get-in network
                                                    [:compute-graph :buffers
                                                     (get parameter :buffer-id)])]
                                 [key
                                  {:description description
                                   :parameter parameter
                                   :buffer buffer}]))
                             parameter-descriptions)
                        (into {}))
        {:keys [forward buffers]} traversal
        {:keys [incoming id outgoing]} (->> forward
                                            (filter #(= test-layer-id
                                                        (get % :id)))
                                            first)
        incoming-buffers (mapv buffers incoming)
        outgoing-buffers (mapv buffers outgoing)]
    {:network network
     :parameters parameters
     :incoming-buffers incoming-buffers
     :outgoing-buffers outgoing-buffers
     :output (get (first outgoing-buffers) :buffer)
     :input-gradient (get (first incoming-buffers) :gradient)
     :numeric-input-gradient (get (first incoming-buffers) :numeric-gradient)}))

(defn unpack-bound-network
  [context network test-layer-id]
  (unpack-network (-> (compute-binding/save-to-network context network {:save-gradients? true})
                      :network)
                  test-layer-id))

(defn- vec->id-val-pairs
  [data-vec id-stem]
  (if (= (count data-vec) 1)
    [[id-stem (first data-vec)]]
    (map-indexed (fn [idx data-item]
                   ;;I know that id generation throughout cortex starts at 1, not 0.
                   [(keyword (format "%s-%s" (name id-stem) (inc idx)))
                    data-item])
                 data-vec)))

(defn vec->stream-map
  "Generate stream id's if the input is not already a map."
  [input-vec stem]
  (if (map? input-vec)
    input-vec
    (->> (vec->id-val-pairs input-vec stem)
         (into {}))))

(defn forward-backward
  "Given a bound network traverse forward and backward using these inputs and
  these output-gradients.  This is used for testing that specific input/output-gradient
  pairs give specific results for layers."
  [bound-network context stream->input-map node-id->output-gradient-map]
  ;;We only provide data for the input streams, we don't provide it for the default loss because
  ;;that loss is not used.
  (let [network-streams (set (->> (network/graph-streams bound-network :inference)
                                  (map first)))
        incoming-streams (set (keys stream->input-map))]
    (when-not (every? incoming-streams
                      network-streams)
     (throw (ex-info "Not all input streams have data"
                     {:bound-streams (clojure.set/intersection network-streams incoming-streams)
                      :unbound-streams (clojure.set/difference network-streams incoming-streams)
                      :incoming-streams incoming-streams}))))
  (let [provided-output-nodes (set (keys node-id->output-gradient-map))
        network-output-nodes (set (network/output-node-ids bound-network :training))]
    (when-not (every? provided-output-nodes network-output-nodes)
      (throw (ex-info "Not all output nodes have provided gradients"
                      {:provided-nodes provided-output-nodes
                       :missing-nodes (clojure.set/difference network-output-nodes provided-output-nodes)}))))
  (let [bound-network (compute-binding/traverse context bound-network stream->input-map :forward)]
    (compute-binding/traverse context bound-network node-id->output-gradient-map :backward)))

(defn forward-backward-bound-network
  [network context input-vec output-gradient-vec test-layer-id]
  (let [input-stream (vec->stream-map input-vec :data)
        output-gradient-stream (vec->stream-map output-gradient-vec test-layer-id)
        network (forward-backward network context input-stream output-gradient-stream)]
    (unpack-bound-network context network test-layer-id)))

(defn forward-backward-test-multiple
  "General test for when there may be a network with multiple inputs or outputs."
  [network context batch-size input-vec output-grad-vec]
  (execute/with-compute-context context
    (-> (bind-test-network network context batch-size)
        (forward-backward-bound-network context input-vec output-grad-vec :test))))

(defn concatenate
  [context]
  (execute/with-compute-context context
    (let [batch-size 4
         item-count 5
         num-inputs 2
         ;;sums to zero
         inputs (->> (partition (* item-count batch-size)
                                (range (* batch-size item-count
                                          num-inputs)))
                     (map #(partition item-count %))
                     (mapv vec)
                     ((fn [input-vec]
                        {:left (first input-vec)
                         :right (second input-vec)})))
         outputs (->> (partition (* item-count batch-size)
                                 (range (* batch-size item-count
                                           num-inputs)))
                      (map #(partition item-count %))
                      (apply interleave)
                      (partition num-inputs)
                      (mapv #(vec (apply concat %))))
         output-gradients outputs
         input-gradients (vec (map inputs [:left :right]))
         network-lr [(layers/input item-count 1 1 :id :right)
                     (layers/input item-count 1 1 :parents [] :id :left)
                     (layers/concatenate :parents [:left :right]
                                         :id :test)]
         {:keys [incoming-buffers outgoing-buffers]}
         (forward-backward-test-multiple network-lr context batch-size
                                         inputs [output-gradients])]
     (is (m/equals input-gradients
                   (mapv :gradient incoming-buffers)))
     (is (m/equals outputs
                   (get-in outgoing-buffers [0 :buffer])))
     (let [network-rl [(layers/input item-count 1 1 :id :right)
                       (layers/input item-count 1 1 :parents [] :id :left)
                       (layers/concatenate :parents [:right :left]
                                           :id :test)]
           swapped-outputs (->> (partition (* item-count batch-size)
                                           (range (* batch-size item-count
                                                     num-inputs)))
                                (map #(partition item-count %))
                                (apply interleave)
                                (partition num-inputs)
                                (mapv #(vec (->> %
                                                 reverse
                                                 (apply concat)))))
           output-gradients swapped-outputs
           {:keys [incoming-buffers outgoing-buffers]}
           (forward-backward-test-multiple network-rl context batch-size
                                           inputs [output-gradients])

           input-gradients (vec (map inputs [:right :left]))]
       (is (m/equals input-gradients
                     (mapv :gradient incoming-buffers)))
       (is (m/equals swapped-outputs
                     (get-in outgoing-buffers [0 :buffer])))))))
