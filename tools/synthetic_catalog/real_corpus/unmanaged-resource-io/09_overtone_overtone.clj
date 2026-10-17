(ns overtone.helpers.zip
  "Useful zip manipulation fns"
  {:author "Sam Aaron"}
  (:require
   [clojure.java.io :as io]
   [clojure.java.io :refer [file]]
   [overtone.helpers.file :refer :all])
  (:import
   (java.io StringWriter FileInputStream FileOutputStream)
   (java.util.zip ZipFile ZipEntry ZipInputStream)))

(defn zip-file
  "Returns an open java.util.zip.ZipFile object representing the zip file
  pointed to by path."
  [path]
  (let [path (resolve-tilde-path path)
        file (file path)]
    (ZipFile. file)))

(defn unzip
  "Unzip a zip file pointed to by zip-path into dest-path dir. Does not allow
  zip file content paths to contain .. parent shortcut for security reasons i.e.
  all unzipped files will be extracted beneath dest-path. If zip file contains
  compressed subdirectories, these will be created too."
  [zip-path dest-path]
  (let [zip-path  (resolve-tilde-path zip-path)
        dest-path (resolve-tilde-path dest-path)
        dest-path (canonical-path dest-path)]
    (when-not (dir-exists? dest-path)
      (throw (Exception. (str "Destination directory does not exist: " dest-path))))
    (when-not (file-exists? zip-path)
      (throw (Exception. (str "Source zip file does not exist: " zip-path))))

    (let [^ZipFile zip (zip-file zip-path)
          entries (.entries zip)
          entries (doall (enumeration-seq entries))]
      (dorun
       (map
        (fn [^java.util.zip.ZipEntry entry]
          (let [name           (.getName entry)
                full-dest-path (mk-path dest-path name)
                ^java.lang.String full-dest-path (canonical-path full-dest-path)]
            (when-not (subdir? full-dest-path dest-path)
              (throw (Exception. "Security warning - unzip was requested to create a path which is not within original dest-path. Aborting operation.")))
            (if (.isDirectory entry)
              (mkdir-p! full-dest-path)
              (let [is (.getInputStream zip entry)
                    fs (FileOutputStream. full-dest-path)]
                (io/copy is fs)))))
        entries)))))
