(ns vmfest.virtualbox.host
  (:import [org.virtualbox_4_3
            HostNetworkInterfaceType Holder IHost IHostNetworkInterface
            IVirtualBox IDHCPServer])
  (:require [vmfest.virtualbox.conditions :as conditions]
            [clojure.tools.logging :as log])
  (:use [vmfest.virtualbox.virtualbox :only [find-dhcp-by-interface-name
                                             create-dhcp-server]]))

(defn- ip-string-to-vec [ip]
  (vec (map #(Integer/parseInt %)
            (clojure.string/split ip #"\."))))
