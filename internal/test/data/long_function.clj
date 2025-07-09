(ns long-function)

;; Example 1: Function with 59 lines
(defn validate-user1
  [user]
  (let [errors (atom [])]

    (if (string? (:name user))
      (do
        (when (< (count (:name user)) 2)
          (swap! errors conj "Name is too short."))
        (when (> (count (:name user)) 100)
          (swap! errors conj "Name is too long."))
        (when (clojure.string/blank? (:name user))
          (swap! errors conj "Name cannot be blank."))
        (when (re-find #"\d" (:name user))
          (swap! errors conj "Name must not contain numbers.")))
      (swap! errors conj "Invalid name."))

    (let [email (:email user)]
      (if (string? email)
        (do
          (if (not (re-matches #".+@.+\..+" email))
            (swap! errors conj "Invalid email format."))
          (if (or (clojure.string/includes? email "spam")
                  (clojure.string/includes? email "fake"))
            (swap! errors conj "Email contains suspicious terms."))
          (when (clojure.string/ends-with? email ".xyz")
            (swap! errors conj "Emails ending in .xyz are not allowed.")))
        (swap! errors conj "Email must be a string.")))

    (let [pwd (:password user)]
      (if (string? pwd)
        (do
          (when (not-any? #(Character/isUpperCase %) pwd)
            (swap! errors conj "Password must contain an uppercase letter."))
          (when (not-any? #(Character/isDigit %) pwd)
            (swap! errors conj "Password must contain a number."))
          (when (not-any? #(contains? #{\! \@ \# \$ \% \&} %) pwd)
            (swap! errors conj "Password must contain a special character."))
          (when (clojure.string/includes? pwd " ")
            (swap! errors conj "Password must not contain spaces.")))
        (swap! errors conj "Invalid password.")))

    (let [age (:age user)]
      (cond
        (nil? age) (swap! errors conj "Age is required.")
        (not (number? age)) (swap! errors conj "Age must be a number.")
        (> age 120) (swap! errors conj "Invalid age.")))

    (let [prefs (:preferences user)]
      (if (sequential? prefs)
        (do
          (when (empty? prefs)
            (swap! errors conj "Preferences list is empty."))
          (when (> (count prefs) 20)
            (swap! errors conj "Too many preferences.")))
        (swap! errors conj "Preferences must be a list.")))

    (if (empty? @errors)
      {:valid true}
      {:valid false :errors @errors})))


;; Example 2: Function with 58 lines
(defn validate-user2
  [user]
  (let [errors (atom [])]

    (if (string? (:name user))
      (do
        (when (< (count (:name user)) 2)
          (swap! errors conj "Name is too short."))
        (when (> (count (:name user)) 100)
          (swap! errors conj "Name is too long."))
        (when (clojure.string/blank? (:name user))
          (swap! errors conj "Name cannot be blank."))
        (when (re-find #"\d" (:name user))
          (swap! errors conj "Name must not contain numbers.")))
      (swap! errors conj "Invalid name."))

    (let [email (:email user)]
      (if (string? email)
        (do
          (if (not (re-matches #".+@.+\..+" email))
            (swap! errors conj "Invalid email format."))
          (if (or (clojure.string/includes? email "spam")
                  (clojure.string/includes? email "fake"))
            (swap! errors conj "Email contains suspicious terms."))
          (when (clojure.string/ends-with? email ".xyz")
            (swap! errors conj "Emails ending in .xyz are not allowed.")))
        (swap! errors conj "Email must be a string.")))

    (let [pwd (:password user)]
      (if (string? pwd)
        (do
          (when (not-any? #(Character/isUpperCase %) pwd)
            (swap! errors conj "Password must contain an uppercase letter."))
          (when (not-any? #(Character/isDigit %) pwd)
            (swap! errors conj "Password must contain a number."))
          (when (not-any? #(contains? #{\! \@ \# \$ \% \&} %) pwd)
            (swap! errors conj "Password must contain a special character."))
          (when (clojure.string/includes? pwd " ")
            (swap! errors conj "Password must not contain spaces.")))
        (swap! errors conj "Invalid password.")))

    (let [age (:age user)]
      (cond
        (not (number? age)) (swap! errors conj "Age must be a number.")
        (> age 120) (swap! errors conj "Invalid age.")))

    (let [prefs (:preferences user)]
      (if (sequential? prefs)
        (do
          (when (empty? prefs)
            (swap! errors conj "Preferences list is empty."))
          (when (> (count prefs) 20)
            (swap! errors conj "Too many preferences.")))
        (swap! errors conj "Preferences must be a list.")))

    (if (empty? @errors)
      {:valid true}
      {:valid false :errors @errors})))


;; ========== CASES THAT SHOULD NOT BE DETECTED ==========


;; Example 3: Function with 57 lines
(defn validate-user3
  [user]
  (let [errors (atom [])]

    (if (string? (:name user))
      (do
        (when (< (count (:name user)) 2)
          (swap! errors conj "Name is too short."))
        (when (> (count (:name user)) 100)
          (swap! errors conj "Name is too long."))
        (when (clojure.string/blank? (:name user))
          (swap! errors conj "Name cannot be blank."))
        (when (re-find #"\d" (:name user))
          (swap! errors conj "Name must not contain numbers.")))
      (swap! errors conj "Invalid name."))

    (let [email (:email user)]
      (if (string? email)
        (do
          (if (not (re-matches #".+@.+\..+" email))
            (swap! errors conj "Invalid email format."))
          (if (or (clojure.string/includes? email "spam")
                  (clojure.string/includes? email "fake"))
            (swap! errors conj "Email contains suspicious terms."))
          (when (clojure.string/ends-with? email ".xyz")
            (swap! errors conj "Emails ending in .xyz are not allowed.")))
        (swap! errors conj "Email must be a string.")))

    (let [pwd (:password user)]
      (if (string? pwd)
        (do
          (when (not-any? #(Character/isUpperCase %) pwd)
            (swap! errors conj "Password must contain an uppercase letter."))
          (when (not-any? #(Character/isDigit %) pwd)
            (swap! errors conj "Password must contain a number."))
          (when (not-any? #(contains? #{\! \@ \# \$ \% \&} %) pwd)
            (swap! errors conj "Password must contain a special character."))
          (when (clojure.string/includes? pwd " ")
            (swap! errors conj "Password must not contain spaces.")))
        (swap! errors conj "Invalid password.")))

    (let [age (:age user)]
      (cond
        (not (number? age)) (swap! errors conj "Age must be a number.")))

    (let [prefs (:preferences user)]
      (if (sequential? prefs)
        (do
          (when (empty? prefs)
            (swap! errors conj "Preferences list is empty."))
          (when (> (count prefs) 20)
            (swap! errors conj "Too many preferences.")))
        (swap! errors conj "Preferences must be a list.")))

    (if (empty? @errors)
      {:valid true}
      {:valid false :errors @errors})))


;; Example 4: Function with 15 lines
(defn validate-user4
  [user]
  (let [errors (atom [])]

    (if (string? (:name user))
      (do
        (when (< (count (:name user)) 2)
          (swap! errors conj "Name is too short."))
        (when (> (count (:name user)) 100)
          (swap! errors conj "Name is too long."))
        (when (clojure.string/blank? (:name user))
          (swap! errors conj "Name cannot be blank."))
        (when (re-find #"\d" (:name user))
          (swap! errors conj "Name must not contain numbers.")))
      (swap! errors conj "Invalid name."))))