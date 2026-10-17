(ns metabase.util.files
  "Low-level file-related functions for implementing Metabase plugin functionality. These use the `java.nio.file`
  library rather than the usual `java.io` stuff because it abstracts better across different filesystems (such as
  files in a normal directory vs files inside a JAR.)

  As much as possible, this namespace aims to abstract away the `nio.file` library and expose a set of high-level
  *file-manipulation* functions for the sorts of operations the plugin system needs to perform."
  (:require
   [babashka.fs :as fs]
   [clojure.java.io :as io]
   [clojure.string :as str]
   [metabase.util :as u]
   [metabase.util.i18n :refer [trs]]
   [metabase.util.log :as log])
  (:import
   (java.io FileNotFoundException)
   (java.net URL)
   (java.nio.file CopyOption Files FileSystem FileSystemAlreadyExistsException
                  FileSystems LinkOption OpenOption Path Paths StandardCopyOption)
   (java.nio.file.attribute FileAttribute)
   (java.util Collections)
   (java.util.zip ZipInputStream)))

(defn- jar-file-system-from-url ^FileSystem [^URL url]
  (let [uri (.toURI url)]
    (try
      (FileSystems/newFileSystem uri Collections/EMPTY_MAP)
      (catch FileSystemAlreadyExistsException _
        (log/info "File system at" uri "already exists")
        (FileSystems/getFileSystem uri)))))

(defn running-from-jar?
  "Returns true iff we are running from a jar.

  .getResource will return a java.net.URL, and those start with \"jar:\" if and only if the app is running from a jar.

  More info: https://docs.oracle.com/en/java/javase/11/docs/api/java.base/java/lang/Thread.html"
  []
  (= "jar" (.. (Thread/currentThread)
               getContextClassLoader
               (getResource ".keep-me")
               getProtocol)))

(defn get-jar-path
  "Returns the path to the currently running jar file.

  More info: https://stackoverflow.com/questions/320542/how-to-get-the-path-of-a-running-jar-file"
  []
  (assert (running-from-jar?) "Can only get-jar-path when running from a jar.")
  (-> (class {})
      (.getProtectionDomain)
      (.getCodeSource)
      (.getLocation)
      (.toURI) ;; avoid problems with special characters in path.
      (.getPath)))

(defn find-in-current-jar
  "Find matching files in the current jar. See
  https://docs.oracle.com/javase/8/docs/api/java/nio/file/FileSystem.html#getPathMatcher-java.lang.String- to
  understand the syntax of [[pattern]].  Motivating use case is:

  `(u.files/find-in-current-jar \"glob:/metabase/*/metabase-plugin.yaml\")`

  to find the plugin manifests in the jar."
  [pattern]
  (let [jar-url (URL. (str "jar:file:" (get-jar-path) "!/"))]
    (with-open [fs (jar-file-system-from-url jar-url)]
      (let [matcher (.getPathMatcher fs pattern)
            root    (.getPath fs "/" (into-array String []))
            files   (atom [])]
        (Files/walkFileTree root
                            (proxy [java.nio.file.SimpleFileVisitor] []
                              (preVisitDirectory [dir _attrs]
                                (if (or (str/starts-with? (str dir) "/metabase")
                                        (= (str "/") (str dir)))
                                  java.nio.file.FileVisitResult/CONTINUE
                                  java.nio.file.FileVisitResult/SKIP_SUBTREE))
                              (visitFile [file _attrs]
                                (when (.matches matcher file)
                                  (swap! files conj file))
                                java.nio.file.FileVisitResult/CONTINUE)))
        (into [] (map #(.toUri ^Path %) @files))))))
