(ns multiple-evaluation-in-macros-catch-clauses)

;; Only one catch clause handles a given exception.
(defmacro alt-catches [x f]
  `(try (~f)
        (catch IllegalStateException e# (println ~x))
        (catch Exception e# (println ~x))))

;; The try body evaluates x and a later exception re-evaluates it in catch.
(defmacro body-and-catch [x f]
  `(try (println ~x) (~f)
        (catch Exception e# (println ~x))))
