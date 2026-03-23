;; Not from source
(def global-state
  (atom {:ui-state  {:theme :light}
         :history (atom [])}))

;; Anti-pattern: Ref dentro de Atom
(def system-state
  (atom {:config (ref {:max-connections 10})
         :status :ok}))

;; Atualizando a ref interna
(dosync
  (alter (:config @system-state) assoc :max-connections 20))

;; O atom externo não “sabe” que a ref mudou instantaneamente

(def test-cases
  [(atom {:inner-atom (atom 0)})
   (atom {:inner-ref (ref 100)})
   (atom {:inner-volatile (volatile! 42)})])
