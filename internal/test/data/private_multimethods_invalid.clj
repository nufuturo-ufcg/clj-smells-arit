(ns private-multimethods-invalid)

(defn- malformed-definitions []
  (defmulti only-name)
  (defmethod only-name :kind))
