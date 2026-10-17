(ns vmfest.virtualbox.machine-config
  (:use [slingshot.slingshot :only [throw+ try+]]
        [vmfest.virtualbox.image :only [ make-immutable]]
        [vmfest.virtualbox.host :only [add-host-only-interface]])
  (:require [vmfest.virtualbox.machine :as machine]
            [vmfest.virtualbox.enums :as enums]
            [vmfest.virtualbox.model :as model]
            [clojure.tools.logging :as log]
            [vmfest.virtualbox.virtualbox :as vbox]
            [vmfest.virtualbox.conditions :as conditions]
            [vmfest.virtualbox.image :as image])
  (:import [org.virtualbox_4_3 AccessMode VBoxException NetworkAttachmentType
            HostNetworkInterfaceType INetworkAdapter IMedium IMachine
            IHostNetworkInterface INATEngine]))

(def controller-type-checkers
  {:scsi #{:lsi-logic :bus-logic}
   :ide #{:piix3 :piix4 :ich6}
   :sata #{:intel-ahci}
   :sas #{:lsi-logic-sas}})

(defn check-controller-type [bus controller-type]
  ((bus controller-type-checkers) controller-type))

(defn add-storage-controller
  [^IMachine m ^String name storage-bus-key & [controller-type-key]]
  (when (and controller-type-key
             (not (check-controller-type storage-bus-key controller-type-key)))
    (throw+ {:type :machine-builder
                 :message
                 (format "Bus of type %s is not compatible with controller %s"
                         storage-bus-key
                         controller-type-key)}))
  (let [storage-bus (enums/key-to-storage-bus storage-bus-key)
        controller-type (when controller-type-key
                          (enums/key-to-storage-controller-type
                           controller-type-key))]
    (log/debugf "add-storage-controller: m=%s name='%s' storage-bus %s"
                m name storage-bus)
    (let [sc (.addStorageController m name storage-bus)]
      (when controller-type
        (.setControllerType sc controller-type)))))
