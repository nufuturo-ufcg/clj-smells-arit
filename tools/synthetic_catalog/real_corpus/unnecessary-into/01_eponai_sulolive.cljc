(ns eponai.common.photos
  (:require
    #?(:clj [clj-http.client :as http]
       :cljs [cljs-http.client :as http])
            [clojure.string :as string]
            [taoensso.timbre :refer [debug]]
            [eponai.common.shared :as shared]))

(def transformations
  {:transformation/micro           "micro"
   :transformation/thumbnail-tiny  "thumbnail-tiny"
   :transformation/thumbnail       "thumbnail"
   :transformation/thumbnail-large "thumbnail-large"
   :transformation/preview         "preview"
   :transformation/cover           "cover"
   :transformation/banner          "city-banner"})

(def storage-host "https://res.cloudinary.com/sulolive")

(defn transformation-param
  "Get transformation parameter for URL given the key"
  [k]
  (when k
    (str "t_" (get transformations k))))

(defn transform [public-id & [transformation file-ext]]
  (let [ext (or file-ext "jpg")
        t (transformation-param transformation)
        url (string/join "/" (into [] (remove nil? [storage-host "image/upload" t (str public-id "." ext)])))]
    url))
