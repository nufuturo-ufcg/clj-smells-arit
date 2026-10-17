(ns keycloak.starter
  (:gen-class)
  (:require
   [clojure.edn :as edn]
   [clojure.java.io :as io]
   [clojure.string :as str]
   [clojure.pprint :as pprint]

   [environ.core :as environ]
   [jsonista.core :as json]
   [me.raynes.fs :as fs]
   [sci.core :as sci]
   [talltale.core :as talltale :refer :all]
   [cli-matic.core :refer [run-cmd]]
   [clj-yaml.core :as yaml]

   [keycloak.admin :as admin :refer :all]
   [keycloak.deployment :as deployment :refer [client-conf keycloak-client]]
   [keycloak.meta :as meta]
   [keycloak.user :as user]
   [keycloak.utils :as utils :refer [list-files]]
   [keycloak.vault.protocol :as vault :refer [Vault write-secret!]]
   [keycloak.vault.hashicorp :as hashicorp-vault]
   [keycloak.vault.google :as google-vault]
   [keycloak.reconciliation :as reconciliation]
   [clojure.pprint :as pp])
  (:import java.io.PushbackReader))

(defn generate-user [username-creator-fn role group subgroup idx & opts]
  (merge (talltale/person)
         {:username (apply username-creator-fn role group subgroup idx opts)
          :password "password"}))

(defn gen-users! [^org.keycloak.admin.client.Keycloak admin-client realm-name {:keys [groups] :as data}]
  (when (and (:generated-users-by-group-and-role data) (> 0 (:generated-users-by-group-and-role data)))
    (doseq [{:keys [name subgroups]} groups]
      (let [^org.keycloak.representations.idm.GroupRepresentation group (admin/get-group admin-client realm-name (get-group-id admin-client realm-name name))]
        (doseq [[idx {subgroup-name :name attributes :attributes}] (map-indexed vector subgroups)]
          (let [^org.keycloak.representations.idm.GroupRepresentation subgroup (admin/get-subgroup admin-client realm-name (.getId group) (get-subgroup-id admin-client realm-name (.getId group) subgroup-name))]
            (doseq [role (:roles data)]
              (doseq [i (range 1 (inc (:generated-users-by-group-and-role data)))]
                (let [user         (generate-user (:username-creator-fn data) role (.getName group) (.getName subgroup) i)
                      created-user (user/create-or-update-user! admin-client realm-name user [role] nil)]
                  (println (format "      User \"%s\" created with realm-roles %s and client-roles %s" (:username user) [role] nil))
                  (println (format "      Add user \"%s\" to group \"%s\"" (:username user) (.getName subgroup)))
                  (add-user-to-group! admin-client realm-name (.getId subgroup) (.getId created-user)))))))))))
