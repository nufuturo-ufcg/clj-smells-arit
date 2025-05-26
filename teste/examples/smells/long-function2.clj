(defn process-users [users]
  (let [filtered (filter #(> (:age %) 18) users)
        formatted (map #(str (:first-name %) " " (:last-name %)) filtered)
        with-age (map #(hash-map :full-name %1 :age (:age %2)) formatted filtered)
        with-status (map #(assoc % :status (if (> (:age %) 60) "senior" "adult")) with-age)
        with-id (map-indexed #(assoc %2 :id %1) with-status)
        with-email (map #(assoc % :email (str (clojure.string/lower-case (:full-name %)) "@example.com")) with-id)
        with-created-at (map #(assoc % :created-at (java.time.LocalDateTime/now)) with-email)
        with-updated-at (map #(assoc % :updated-at (java.time.LocalDateTime/now)) with-created-at)
        with-active (map #(assoc % :active true) with-updated-at)
        with-roles (map #(assoc % :roles ["user"]) with-active)
        with-preferences (map #(assoc % :preferences {:theme "light" :language "en"}) with-roles)
        with-settings (map #(assoc % :settings {:notifications true :two-factor false}) with-preferences)
        with-metadata (map #(assoc % :metadata {:browser "unknown" :platform "web"}) with-settings)
        with-stats (map #(assoc % :stats {:login-count 0 :last-login nil}) with-metadata)
        with-profile (map #(assoc % :profile {:bio "" :avatar nil}) with-stats)
        with-permissions (map #(assoc % :permissions ["read" "write"]) with-profile)
        final-users (map #(assoc % :version 1) with-permissions)]
    final-users))

(println (process-users [{:first-name "Alice" :last-name "Smith" :age 22}
                        {:first-name "Bob" :last-name "Johnson" :age 17}
                        {:first-name "Charlie" :last-name "Brown" :age 25}])) 