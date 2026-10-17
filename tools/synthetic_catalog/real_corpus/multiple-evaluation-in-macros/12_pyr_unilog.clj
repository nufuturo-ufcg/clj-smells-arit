(ns unilog.context
  "Provide a way to interact with Mapped Diagnostic Context")

(defn push-context
  "Add a key to the current Mapped Diagnostic Context"
  [k v]
  (org.slf4j.MDC/put (name k) (str v)))

(defn set-context
  "Sets the current Mapped Diagnostic Context"
  [ctx]
  (org.slf4j.MDC/setContextMap ctx))

(defmacro with-context
  "Execute body with the Mapped Diagnostic Context updated from
   keys found in the ctx map."
  [ctx & body]
  `(if-not (map? ~ctx)
     (throw (ex-info "with-context expects a map" {}))
     (let [copy# (org.slf4j.MDC/getCopyOfContextMap)]
       (try
         (doseq [[k# v#] ~ctx]
           (push-context k# v#))
         ~@body
         (finally
           (set-context (or copy# {})))))))
