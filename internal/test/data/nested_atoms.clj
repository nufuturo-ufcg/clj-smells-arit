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

;; A local atom being updated is not evidence that it is stored in another atom.
(defn update-local-state []
  (let [state (atom {})]
    (swap! state assoc :ready true)
    @state))

;; Neither is a local atom coexisting with an unrelated outer update.
(def registry (atom {}))
(defn unrelated-update []
  (let [local-state (atom {})]
    (swap! local-state assoc :ready true)
    (swap! registry assoc :status :ok)))

;; The reference itself is certainly inserted into registry.
(defn register-nested-state []
  (let [local-state (atom {})]
    (swap! registry assoc :worker local-state)))

;; Shadowing core atom must not be interpreted as reference creation.
(defn shadowed-atom [atom]
  (atom {:value (atom 1)}))
