(ns misuse-of-dynamic-scope)

(def ^:dynamic *compiler-mode* :normal)
(binding [*compiler-mode* :fast]
  (println *compiler-mode*))

(future
  (binding [*request-context* {:id 1}]
    (println *request-context*)))

(future
  (def *runtime-context* {}))

(future
  (binding [*request-context* nil]
    (println *request-context*)))

(future
  (binding [*read-eval* false]
    (println *read-eval*)))

;; bound-fn explicitly propagates dynamic bindings to the generated function.
(bound-fn []
  (binding [*request-context* {:id 2}]
    (println *request-context*)))

;; A binding inside a core.async state machine can be interrupted at a park
;; point and must not be treated like ordinary worker-local future setup.
(go
  (binding [*worker-context* :go]
    (println *worker-context*)))
