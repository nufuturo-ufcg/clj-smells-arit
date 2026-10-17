(ns appengine-magic.services.datastore
  (:use appengine-magic.utils)
  (:import [com.google.appengine.api.datastore DatastoreService DatastoreServiceFactory
            DatastoreServiceConfig DatastoreServiceConfig$Builder
            EntityNotFoundException
            ReadPolicy ReadPolicy$Consistency ImplicitTransactionManagementPolicy
            Key KeyFactory
            Entity
            FetchOptions$Builder
            Query Query$FilterOperator Query$SortDirection]
           ;; types
           [com.google.appengine.api.datastore Blob ShortBlob Text Link]
           com.google.appengine.api.blobstore.BlobKey))

(defn get-datastore-service []
  (when (nil? @*datastore-service*)
    (reset! *datastore-service* (DatastoreServiceFactory/getDatastoreService)))
  @*datastore-service*)

(defmacro with-transaction [& body]
  `(binding [*current-transaction* (.beginTransaction (get-datastore-service))]
     (try
       (let [body-result# (do ~@body)]
         (.commit *current-transaction*)
         body-result#)
       (catch Throwable err#
         (do (.rollback *current-transaction*)
             (throw err#))))))
