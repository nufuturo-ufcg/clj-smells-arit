(ns korma.db
  "Functions for creating and managing database specifications."
  (:require [clojure.java.jdbc :as jdbc]
            [clojure.string]))

(def ^:dynamic *current-db* nil)

(def ^:dynamic *current-conn* nil)

(defn- exec-sql [{:keys [results sql-str params]}]
  (let [keys (get-in *current-db* [:options :naming :keys])
        sql-params (apply vector sql-str params)]
    (case results
      :results (jdbc/query *current-conn* sql-params {:identifiers keys})
      :keys (jdbc/db-do-prepared-return-keys *current-conn* sql-params)
      (jdbc/db-do-prepared *current-conn* sql-params))))
