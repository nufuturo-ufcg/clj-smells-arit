(ns gen-ci
  (:require
   [babashka.tasks :as tasks]
   [clj-yaml.core :as yaml]
   [clojure.string :as str]
   [flatland.ordered.map :refer [ordered-map]]))

(def graalvm-version "25")

(defn run
  ([cmd-name cmd]
   (run cmd-name cmd nil))
  ([cmd-name cmd no-output-timeout]
   (let [base {:run {:name    cmd-name
                     :command cmd}}]
     (if no-output-timeout
       (assoc-in base [:run :no_output_timeout] no-output-timeout)
       base))))

(defn gen-steps
  [shorted? steps]
  (if shorted?
    [(run "Shorted" "echo 'Skipping Run'")]
    steps))

(defn gen-job
  [shorted? conf]
  (if shorted?
    (-> conf
        (dissoc :machine :macos)
        (assoc :resource_class "small" :docker [{:image "ubuntu:latest"}]))
    conf))

(defn pull-submodules
  []
  (run "Pull Submodules" "git submodule init\ngit submodule update"))

(defn docker
  [shorted?]
  (gen-job
   shorted?
   (ordered-map
    :machine {:image "ubuntu-2004:2024.05.1"}
    :steps
    (gen-steps
     shorted?
     [:checkout
      (pull-submodules)
      "setup-docker-buildx"
      {:attach_workspace {:at "/tmp"}}
      (run "Extract bb from release"
        "VERSION=$(cat resources/BABASHKA_VERSION)
tar xzf /tmp/release/babashka-$VERSION-linux-amd64.tar.gz -C .")
      {:run
       {:name        "Build Docker image"
        :environment {:PLATFORMS "linux/amd64,linux/arm64"}
        :command     "./bb .circleci/script/docker.clj"}}]))))

(defn unix
  [shorted? static? musl? arch executor-conf resource-class graalvm-home platform]
  (let [env              {:LEIN_ROOT         "true"
                          :GRAALVM_VERSION   graalvm-version
                          :GRAALVM_HOME      graalvm-home
                          :BABASHKA_PLATFORM (if (= "mac" platform)
                                               "macos"
                                               platform)
                          :BABASHKA_TEST_ENV "native"
                          :BABASHKA_XMX      "-J-Xmx6500m"
                          :BABASHKA_SHA (System/getenv "CIRCLE_SHA1")}
        env              (if (= "aarch64" arch)
                           (assoc env :BABASHKA_ARCH arch)
                           env)
        env              (if static?
                           (assoc env :BABASHKA_STATIC "true")
                           env)
        env              (if musl?
                           (assoc env :BABASHKA_MUSL "true")
                           env)
        env              (if (= "mac" platform)
                           (assoc env :MACOSX_DEPLOYMENT_TARGET 10.13)
                           env)
        base-install-cmd "sudo apt-get update\nsudo apt-get -y install build-essential zlib1g-dev"
        cache-key        (format "%s-%s{{ checksum \"project.clj\" }}-{{ checksum \".circleci/config.yml\" }}"
                                 platform
                                 (if (= "aarch64" arch)
                                   "aarch64-"
                                   ""))]
    (gen-job shorted?
             (merge
              executor-conf
              (ordered-map
               :working_directory "~/repo"
               :environment env
               :resource_class resource-class
               :steps (gen-steps shorted?
                                 (filter some?
                                         [:checkout
                                          (when (contains? #{"linux" "linux-aarch64"} platform)
                                            (run "Check max glibc version" "script/check_glibc.sh"))
                                          {:attach_workspace {:at "/tmp"}}
                                          (run "Pull Submodules" "git submodule init\ngit submodule update")
                                          {:restore_cache
                                           {:keys [cache-key]}}
                                          (when (= "mac" platform)
                                            (run "Install Rosetta" "sudo /usr/sbin/softwareupdate --install-rosetta --agree-to-license"))
                                          (run "Install Clojure" "sudo script/install-clojure")
                                          (when (= "mac" platform)
                                            (run "Install Leiningen" "script/install-leiningen"))
                                          (when (not= "mac" platform)
                                            (run "Install native dev tools"
                                              (if (and static? musl? (not= "aarch64" arch))
                                                (str base-install-cmd "\nsudo -E script/setup-musl")
                                                base-install-cmd)))
                                          (run "Download GraalVM" "script/install-graalvm")
                                          #_(run "Download iprof" "curl -sLO 'https://github.com/babashka/pgo-profiles/releases/download/2023.10.11/default.iprof'")
                                          (run "Build binary" (if (= "aarch64" arch)
                                                                "script/uberjar\nscript/compile -H:PageSize=64K # --pgo=default.iprof"
                                                                "script/uberjar\nscript/compile # --pgo=default.iprof") "30m")
                                          (run "Release" ".circleci/script/release")
                                          {:persist_to_workspace {:root  "/tmp"
                                                                  :paths ["release"]}}
                                          (run "Run tests" "script/test\nscript/run_lib_tests")
                                          (run "Release + publish"
                                            (str/join "\n" ["export BABASHKA_RELEASE=true"
                                                            ".circleci/script/release"]))
                                          {:save_cache
                                           {:paths ["~/.m2" "~/graalvm"]
                                            :key   cache-key}}
                                          {:store_artifacts {:path        "/tmp/release"
                                                             :destination "release"}}
                                          (run "Publish artifact link to Slack"
                                            "./bb .circleci/script/publish_artifact.clj || true")])))))))
