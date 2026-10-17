(ns com.rpl.agent-o-rama.impl.ui.rpc.datasets
  (:require
   [com.rpl.agent-o-rama :as aor]
   [com.rpl.agent-o-rama.impl.types :as aor-types]
   [com.rpl.agent-o-rama.impl.ui.handlers.common :as common]
   [com.rpl.agent-o-rama.impl.queries :as queries]
   [com.rpl.agent-o-rama.impl.datasets :as datasets]
   [com.rpl.agent-o-rama.impl.helpers :as h]
   [clojure.string :as str]
   [jsonista.core :as j])
  (:use [com.rpl.rama]))

(defn- get-manager [system module-id]
  (get-in system [:aor-cache module-id :manager]))

(defn preview-expression!!
  [system {:keys [module-id dataset-id snapshot-name source-field expression type]}]
  (let [manager (get-manager system module-id)
        {:keys [search-examples-query multi-examples-query]}
        (aor-types/underlying-objects manager)
        search-result (foreign-invoke-query search-examples-query
                                            dataset-id
                                            (when-not (str/blank? snapshot-name) snapshot-name)
                                            {}
                                            1
                                            nil)
        example-summary (first (:examples search-result))]
    (if-not example-summary
      {:status :ok :result nil :error "No examples found in dataset"}
      (let [full-examples (foreign-invoke-query multi-examples-query
                                                dataset-id
                                                (when-not (str/blank? snapshot-name) snapshot-name)
                                                [(:id example-summary)])
            example (get full-examples (:id example-summary))
            source-data (case source-field
                          :input (:input example)
                          :reference-output (:reference-output example)
                          nil)
            preview-result
            (if (nil? source-data)
              ::no-source-data
              (try
                (case type
                  :path
                  (h/read-json-path source-data expression)
                  :template
                  (let [template-obj
                        (if (string? expression)
                          (if (or (str/starts-with? (str/trim expression) "{")
                                  (str/starts-with? (str/trim expression) "["))
                            (j/read-value expression)
                            expression)
                          expression)]
                    (h/resolve-json-path-template template-obj source-data)))
                (catch Exception e
                  (let [msg (or (.getMessage e) "")]
                    (cond
                      (or (str/includes? msg "can not be null")
                          (str/includes? msg "cannot be null")
                          (str/includes? msg "No results")
                          (str/includes? msg "Missing property"))
                      ::path-not-found
                      :else
                      (throw e))))))]
        (cond
          (= preview-result ::no-source-data)
          {:status :ok :result nil :error (str "No " (name source-field) " data in this example")}
          (= preview-result ::path-not-found)
          {:status :ok :result nil :error "Path not found in data"}
          :else
          {:status :ok
           :result (common/->ui-serializable preview-result)
           :example-preview (common/->ui-serializable source-data)})))))
