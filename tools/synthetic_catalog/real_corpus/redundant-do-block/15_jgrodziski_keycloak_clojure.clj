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

(defn export-json [export-dir export-file-without-extension client-id path secrets]
  (let [secrets-file (clojure.java.io/file export-dir (str export-file-without-extension ".json"))]
    (fs/touch secrets-file)
    (println "secrets" secrets)
    (json/write-value secrets-file secrets)
    (println (format "Secret of client \"%s\" exported in files %s at path %s" client-id (str secrets-file) path))))

(defn export-yaml [export-dir export-file-without-extension client-id path secrets]
  (let [secrets-file (clojure.java.io/file export-dir (str export-file-without-extension ".yml"))]
    (fs/touch secrets-file)
    (spit secrets-file (yaml/generate-string secrets))
    (println (format "Secret of client \"%s\" exported in files %s at path %s" client-id (str secrets-file) path))))

(defn- env-var-or-dir? [dir]
  (if (System/getenv dir)
    (System/getenv dir)
    dir))

(defn export-secret-in-files! [^org.keycloak.admin.client.Keycloak keycloak-client realm-name client-id {:keys [name-without-extension export-dir path] :as secret-file}]
  (let [export-dir          (or export-dir "/etc/keycloak")
        path                (conj path (keyword client-id) :secret)
        _                   (println (format "Secret of client \"%s\" will be exported at path %s in files %s in directory %s" client-id path (str name-without-extension ".edn|json|yml") export-dir))
        secret              (get-client-secret keycloak-client realm-name client-id)
        secret-file         (or name-without-extension ".keycloak-secrets")
        resolved-export-dir (env-var-or-dir? export-dir)
        secrets-file        (clojure.java.io/file resolved-export-dir (str secret-file ".edn"))
        _                   (fs/mkdir resolved-export-dir)
        _                   (fs/touch secrets-file)
        secrets             (-> (or (clojure.edn/read-string (slurp secrets-file)) {})
                                (assoc-in path secret))]
    (println (format "Secret of client \"%s\" exported in files %s at path %s" client-id (str secrets-file) path))
    (spit secrets-file secrets)
    (export-json resolved-export-dir secret-file client-id path secrets)
    (export-yaml resolved-export-dir secret-file client-id path secrets)
    secrets-file))

(defmulti export-secret-in-vault!
  "take the vendor value to dispatch the call to proper function, other value are specific to vault implementation:
  - `:hashicorp`: in infra-context, [:vault :path] is a string with placeholders as: %1$s is the environment, %2$s is the color, %3$s is the base-domains, %4$s is the client-id (client-id depends of your realm-config.clj code)
  - `:gcp-sm`: the vault entry of infra-contect must contains project-id and secret-id, also the google_application_credentials must be properly defined and available at runtime "
  (fn [_ infra-context _ _] (get-in infra-context [:vault :vendor])))

(defmethod export-secret-in-vault! :hashicorp [^org.keycloak.admin.client.Keycloak keycloak-client {:keys [vault environment color base-domains] :as infra-context} realm-name client-id]
  (let [secret                                  (get-client-secret keycloak-client realm-name client-id)
        {:keys [protocol host port mount path token vendor]} vault
        vault-url                               (hashicorp-vault/vault-url protocol host port)
        vault-client                            (hashicorp-vault/new-client vault-url)
        hashicorp-vault                         (hashicorp-vault/->HashicorpVault vault-client token mount)
        vault-path                              (format path environment color base-domains client-id)]
    (println (format "Secret of client \"%s\" will be exported in hashicorp vault at url %s, mount %s and path %s" client-id vault-url mount vault-path))
    (vault/write-secret! hashicorp-vault vault-path secret)))

(defmethod export-secret-in-vault! :gcp-sm  [^org.keycloak.admin.client.Keycloak keycloak-client {:keys [vault environment color base-domains] :as infra-context} realm-name client-id]
  (let [secret                      (get-client-secret keycloak-client realm-name client-id)
        {:keys [project-id secret-id vendor]} vault
        secret-name                 (format "realm/%s/client/%s" realm-name client-id)
        google-secret-manager       (google-vault/->GoogleSecretManager project-id)]
    (vault/write-secret! google-secret-manager (or secret-id secret-name) secret)
    (println (format "Secret of client \"%s\" will be exported in google secret-manager for project-id %s and secret-id %s secret %s" client-id project-id (or secret-id secret-name) secret))))

(defn generate-user [username-creator-fn role group subgroup idx & opts]
  (merge (talltale/person)
         {:username (apply username-creator-fn role group subgroup idx opts)
          :password "password"}))

(defn init-realm! [^org.keycloak.admin.client.Keycloak admin-client {:keys [name themes login tokens smtp user-admin] :as realm-data}]
  (println (format "Will create realm \"%s\"" name))
  (try (create-realm! admin-client name themes login tokens smtp)
       (println (format "Realm \"%s\" created" name))
       (if-let [user-admin-id (when user-admin (user/user-id admin-client "master" (:username user-admin)))]
         (do
           (println (format "Will update the admin user %s (user-id %s) with %s" (:username user-admin) user-admin-id user-admin))
           (user/update-user! admin-client "master" user-admin-id user-admin))
         (do
           (println (format "Will create the admin users %s with data as %s" (:username user-admin) user-admin))
           (user/create-user! admin-client "master" user-admin)))
       (catch jakarta.ws.rs.ClientErrorException cee
         (if (= (-> cee (.getResponse) (.getStatus)) 409)
           (do
             (try
               (update-realm! admin-client name themes login tokens smtp)
               (println (format "Realm \"%s\" updated" name))
               (catch Throwable t
                 (println (format "Can't update realm %s" name)))))
           (println "Can't create realm " name ", " cee))
         realm-data)
       (catch jakarta.ws.rs.InternalServerErrorException isee
         (println "Can't create realm " isee)
         (try
           (update-realm! admin-client name themes login tokens smtp)
           (println (format "Realm \"%s\" updated" name))
           (catch Throwable t
             (println (format "Can't update realm %s" name))))
         realm-data)
       (catch Exception e
         (println (format "Can't create Realm, exception message %s and realm data:" (.getMessage e)))
         (pp/pprint realm-data)
         realm-data)))

(defn create-mappers! [^org.keycloak.admin.client.Keycloak keycloak-client realm-name client-id mappers]
  (when (and mappers (not (empty? mappers)))
    (println "Create protocol mappers for client" client-id)
    (doall (for [{:keys [name type config] :as mapper} mappers]
             (do (println (format "  Create protocol mapper for client%s: name %s type %s config %s" client-id name type config))
                 (create-protocol-mapper! keycloak-client realm-name client-id (admin/mapper name type config)))))))

(defn init-clients! [^org.keycloak.admin.client.Keycloak admin-client realm-name clients-data infra-context]
  (doseq [{:keys [name public? redirect-uris web-origins mappers] :as client-data} clients-data]
    (let [client (client client-data)
          client-id name
          _ (println (format "Create client \"%s\" in realm %s and client data %s" client-id realm-name client-data))
          created-client (create-or-update-client! admin-client realm-name client)]
      (if (not created-client) (throw (Exception. (format "Client %s not created in realm %s" client-id realm-name))))
      (println (format "Client with Id \"%s\" and clientId \"%s\" created in realm %s" (.getId created-client) (.getClientId created-client) realm-name))
      (create-mappers! admin-client realm-name client-id mappers)
      (when (:secret-file infra-context )
        (export-secret-in-files! admin-client realm-name client-id (:secret-file infra-context)))
      (when (:vault infra-context)
        ;;by default we use hashicorp vault (downward compatibility with previous version)
        (let [infra-context (if (get-in infra-context [:vault :vendor]) infra-context (assoc-in infra-context [:vault :vendor] :hashicorp))]
          (export-secret-in-vault! admin-client infra-context realm-name client-id)))))
  (println (format "%s Clients created in realm %s" (count clients-data) realm-name)))

(defn init-roles! [^org.keycloak.admin.client.Keycloak admin-client realm-name roles-data]
  (doseq [role roles-data]
    (try (create-role! admin-client realm-name role)
         (println (format "Role %s created in realm %s" role realm-name))
         (catch Exception e (get-role admin-client realm-name role)))))

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

(defn init-users! [^org.keycloak.admin.client.Keycloak admin-client realm-name users-data & [opts]]
  (doseq [{:keys [username] :as user} users-data]
    (let [created-user (user/create-or-update-user! admin-client realm-name user (:realm-roles user) (:client-roles user))]
      (println (format "User \"%s\" created with realm-roles %s and client-roles %s" username (pr-str (:realm-roles user)) (:client-roles user)))
      (doseq [subgroup-name (:in-subgroups user)]
        (let [subgroup-id (get-subgroup-id admin-client realm-name (get-group-id admin-client realm-name (:group user)) subgroup-name)]
          (println (format "Add user \"%s\" to group \"%s\"" username subgroup-name))
          (add-user-to-group! admin-client realm-name subgroup-id (.getId created-user)))))))

(defn init!
  "Create a structure of keycloak entities (realm, clients, roles) and fill it with groups and users. Arguments are:
  * `admin-client`: [admin client's _Keycloak_ object](https://www.keycloak.org/docs-api/11.0/javadocs/org/keycloak/admin/client/Keycloak.html) obtained with:  `(keycloak.deployment/keycloak-client (keycloak.deployment/client-conf \"http://localhost:8090\" \"master\"  \"admin-cli\") admin-login admin-password)`
  * `data`: the configuration data with keys `realm`, `roles`, `groups` and `users` (see the documentation for more details https://cljdoc.org/d/keycloak-clojure/keycloak-clojure/1.18.0/doc/administrative-tasks#declarative-creation-of-keycloak-objects)
  * `infra-context`: a description of the keycloak infrastructure, the keys needed here is the :vault for secret export during clients creation
  * `opts`: with two boolean entries `dry-run?` and `apply-deletions?`, respectively not applying the detected steps for attaining the desired state in `data` and whether the steps should delete the entities in Keycloak not in that state (sort of a very strict mode)
  "
([^org.keycloak.admin.client.Keycloak admin-client data infra-context & [opts]]
 (when (or (nil? admin-client) (nil? data))
   (throw (ex-info "Admin client and/or realm config data can't be null")))
 (let [realm-name         (get-in data [:realm :name])
       dry-run?           (:dry-run? opts)
       _                   (println (format "Sync realm %s (dry-run? %s)" realm-name dry-run?))
       config-data-file   (utils/pprint-to-temp-file (str realm-name "-config-data-") data)
       infra-context-file (utils/pprint-to-temp-file (str realm-name "-infra-context-") (update infra-context :keycloak dissoc :password))]
   (println (format "Configuration data can be found at %s" (str config-data-file)))
   (println (format "Infra context can be found at %s" (str infra-context-file)))
   (when (not dry-run?)
     (init-realm!   admin-client (:realm data))
     (init-clients! admin-client realm-name (:clients data) infra-context)
     (init-roles!   admin-client realm-name (:roles data))
     (gen-users!    admin-client realm-name data))
   (reconciliation/reconciliate-groups!        admin-client realm-name (:groups data) opts)
   (reconciliation/reconciliate-users!         admin-client realm-name (:users data)  opts)
   (reconciliation/reconciliate-role-mappings! admin-client realm-name (:roles data)  (:users data) opts)
   ;(init-users!   admin-client realm-name (:users data))
   (when (not dry-run?)
     (println (format "Keycloak realm \"%s\" synchronized" realm-name)))
   data)))

(defn keycloak-auth-server-url [protocol host port path]
  (str protocol "://" host ":" port (or path "/auth")))

(defn- truthy? "Cli-matic value accepted as true" [s]
  (or (= s "Y") (= s "Yes") (= s "On") (= s "T") (= s "True") (= s "1")))

(defn process-args [{:keys [realm-config infra-context] :as args}]
  (let [{:keys [environment color applications vault keycloak secret-file]} infra-context
        {:keys [auth-server-url protocol host port path]}                   (or keycloak args) ;either the params are in the keyclaok config file or each params is passed through a direct param
        login                                                               (or (:login keycloak)    (:login args)    (environ/env :login))
        password                                                            (or (:password keycloak) (:password args) (environ/env :password))
        auth-server-url                                                     (or (when (not-empty (:auth-server-url args)) (:auth-server-url args))
                                                                                (:auth-server-url keycloak)
                                                                                (environ/env :auth-server-url)
                                                                                (keycloak-auth-server-url protocol host port path))
        dry-run?                                                            (or (:dry-run args)         (truthy? (environ/env :dry-run)))
        apply-deletions?                                                    (or (:apply-deletions args) (truthy? (environ/env :apply-deletions)))
        processed-args                                                      {:auth-server-url               auth-server-url
                                                                             :login                         login
                                                                             :keycloak                      keycloak
                                                                             :password                      password
                                                                             :environment                   environment
                                                                             :color                         color
                                                                             :applications                  applications
                                                                             :vault-config                  vault
                                                                             :realm-config                  realm-config
                                                                             :infra-context                 infra-context
                                                                             :dry-run?                      dry-run?
                                                                             :apply-deletions?              apply-deletions?
                                                                             :resources-dir                 (:resources-dir args)
                                                                             :secret-export-dir             (or (get-in infra-context [:secret-file :export-dir]) (:secret-export-dir args))
                                                                             :secret-file-without-extension (or (get-in infra-context [:secret-file :name-without-extension]) (:secret-file-without-extension args))
                                                                             :secret-path                   (or (get-in infra-context [:secret-file :path]))
                                                                             }]
    (when (or (empty? auth-server-url) (empty? password) (empty? login) (nil? realm-config))
      (println "Usage: clj -m keycloak.starter <auth-server-url> <login> <password> <environment> <realm-config> <infra-context>" )
      (throw (ex-info "Usage: clj -m keycloak.starter <auth-server-url> <login> <password> <environment> <realm-config>" processed-args)))
    processed-args))

(defn dissoc-sensitive-data [config-data]
  ;remove password entry of all users
  (update config-data :users (fn [users] (mapv #(update % :password (fn [_] :XXXXXXXX)) users))))

(defn edn-resources-bindings [resources-dir]
  (when resources-dir
    (let [edn-files (utils/list-files resources-dir (fn [f] (= ".edn" (:ext (utils/parse-path f)))))]
      (into {} (map (fn [f]
                      (when f
                        (let [{:keys [name ext]} (utils/parse-path f)
                              edn-content  (when (and (utils/file-exists? f) (utils/file-not-empty? f))
                                             (edn/read (PushbackReader. (io/reader f))))]
                          (when edn-content
                            [(symbol name) (sci/new-var (symbol name) edn-content)])))) edn-files)))))

(defn init-cli! [args]
  (let [{:keys [infra-context resources-dir realm-config secret-export-dir secret-file-without-extension secret-path auth-server-url
                login password environment color applications vault-config keycloak dry-run? apply-deletions?] :as processed-args} (process-args args)]
    (let [admin-client (->  (deployment/client-conf auth-server-url "master" "admin-cli")
                            (deployment/keycloak-client login password))
          sci-environment   (sci/new-var 'environment environment)
          sci-color         (sci/new-var 'color color)
          sci-applications  (sci/new-var 'applications applications)
          sci-keycloak      (sci/new-var 'keycloak keycloak)
          sci-infra-context (sci/new-var 'infra-context infra-context)
          sci-bindings      {:bindings (merge {'environment   sci-environment
                                               'applications  sci-applications
                                               'keycloak      sci-keycloak
                                               'infra-context sci-infra-context
                                               'color         sci-color}
                                              (edn-resources-bindings resources-dir))}
          _                 (println "Execute SCI config with bindings of the resources: " (map str (utils/list-files resources-dir (fn [f] (= ".edn" (:ext (utils/parse-path f)))))))
          ;_                 (println (pr-str (clojure.pprint/pprint sci-bindings)))
          config-data       (sci/eval-string realm-config sci-bindings)]
      (when dry-run?
        (println "Dry run execution of Keycloak-clojure-starter!")
        (println "Infra-context used is:")
        (utils/pprint-to-stdout infra-context)
        ;(println "Keycloak configuration data resulting of script evaluation is:")
        ;(utils/pprint-to-stdout config-data)
        )
      (println (format "Keycloak-clojure-starter version %s targeting Keycloak at %s in env %s with %s realm(s)" meta/version auth-server-url (or environment "localhost") (count config-data)))
      (println (format "Login to %s realm, clientId %s with username %s" "master" "admin-cli" login))
      (if (map? config-data)
        (do
                                        ;(clojure.pprint/pprint (dissoc-sensitive-data config-data))
          (init! admin-client config-data infra-context processed-args))
        (when (or (vector? config-data) (seq? config-data))
          (doseq [realm-data config-data]
            ;(clojure.pprint/pprint (dissoc-sensitive-data realm-data))
            (init! admin-client realm-data infra-context processed-args))))
      (shutdown-agents))))
