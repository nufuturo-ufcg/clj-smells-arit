(ns ethlance.ui.util.tokens
  (:require
    [cljs.core.async :as async]
    [cljs-web3-next.eth :as w3-eth]
    [cljs-web3-next.helpers :as web3-helpers]
    [clojure.math]
    [clojure.walk]
    [district.ui.smart-contracts.queries]
    [district.ui.web3.queries :as web3-queries]
    [ethlance.shared.utils :refer [wei->eth eth->wei]]
    [re-frame.db :refer [app-db]]))

(defn round
  [decimals amount]
  (let [multiplier (clojure.math/pow 10 decimals)]
    (/ (.round js/Math (* multiplier amount)) multiplier)))

(defn human-amount
  [amount token-type & [decimals]]
  (if (not (nil? decimals))
    (round decimals (/ amount (clojure.math/pow 10 decimals)))
    (case (keyword token-type)
      :eth (wei->eth amount)
      amount)))
