(def ^:private version
  (delay (or (System/getenv "DEPS_CLJ_TOOLS_VERSION")
             "1.12.5.1664")))

(def deps-clj-version
  "The current version of deps.clj"
  (-> (io/resource "DEPS_CLJ_VERSION")
      (slurp)
      (str/trim)))
