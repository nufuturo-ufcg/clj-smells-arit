(ns zero-one.geni.utils)

(defn- import-class
  ([cls] (.importClass *ns* (clojure.lang.RT/classForName (str cls))))
  ([pkg cls] (import-class (str pkg \. cls))))

(defmacro with-dynamic-import [imports & body]
  (if (try
        (doall
         (for [imp imports]
           (if (symbol? imp)
             (import-class imp)
             (let [[pkg & classes] imp]
               (doall (for [cls classes] (import-class pkg cls)))))))
        true
        (catch ClassNotFoundException _ nil))
    `(do ~@body :succeeded)
    :failed))
