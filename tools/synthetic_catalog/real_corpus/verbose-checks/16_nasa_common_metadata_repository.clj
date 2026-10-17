(ns cmr.sizing.formats
  (:require
    [clojure.string :as string]
    [cmr.exchange.common.util :as util]
    [taoensso.timbre :as log]))

(defn- get-dimensionality
  "Get the dimensionality, which is the multiplication of all the sizes
  in all the Dimensions. Returns 0 when Dimensions is not present."
  [variable]
  (let [dims (get-in variable [:umm :Dimensions])]
    (if (seq dims)
      (reduce * (map :Size dims))
      0)))

(defn- get-avg-digit-number
  "Returns the average of all the numbers in digit-numbers.
  Return 0 when digit-numbers is empty."
  [digit-numbers]
  (if (seq digit-numbers)
    (/ (reduce + digit-numbers) (count digit-numbers))
    0))

(defn- get-avg-fill-value-digit-number
  "Get the average digit number for each value of FillValues in the variable.
  For example, FillValues containing [{:Value 1} {:Value 100}], returns (1+3)/2=2."
  [variable]
  (let [fvs (get-in variable [:umm :FillValues])
        fvs-digit-numbers (map #(count (str (:Value %))) fvs)]
    (get-avg-digit-number fvs-digit-numbers)))

(defn- process-range
  "Split the range into two parts if the range is from negative to positive.
  For example {:Min -10 :Max 100} becomes [{:Min -10 :Max 0} {:Min 0 :Max 100}]."
  [range]
  (let [min (:Min range)
        max (:Max range)]
    (if (and (< min 0) (> max 0))
      [{:Min min :Max 0} {:Min 0 :Max max}]
      range)))

(defn- get-valid-ranges 
  "Get the ValidRanges from variable. If the range is from negative
  to positive, break it up into two ranges: negative to 0 and 0 to positive."
  [variable]
  (let [vrs (get-in variable [:umm :ValidRanges])]
    (flatten (map process-range vrs))))

(defn- get-avg-valid-range-digit-number
  "Get the average digit number for each range of ValidRanges in the variable.
  For example, ValidRanges containing [{:Min -10 :Max 100} {:Min 10 :Max 1000}] 
  returns (3+1+1+3+2+4)/6=2.3"
  [variable]
  (let [vrs (get-valid-ranges variable)
        vrs-min-max (concat (map :Min vrs) (map :Max vrs))
        vrs-digit-numbers (map #(count (str %)) vrs-min-max)]
    (get-avg-digit-number vrs-digit-numbers)))

(defn- estimate-ascii-size-per-granule
  "Calculate the estimated size for ASCII output per granule, for the variable.
  In ASCII output, the actual size for a granule in bytes is the sum of digit number
  of all the dimension data + 2 bytes(comma and space) for each data except for the last one.
  To estimate, we have the following cases: 
  1. dimensionality <= 0.
     Returns 0.
  2. avg-fill-value-digit-number = 0, avg-valid-range-digit-number != 0
     Returns avg-valid-range-digit-number * dimensionality + 2 * (dimensionality -1)
  3. avg-fill-value-digit-number != 0, avg-valid-range-digit-number = 0
     Returns avg-fill-value-digit-number * dimensionality + 2 * (dimensionality -1)
  4. avg-fill-value-digit-number = 0, avg-valid-range-digit-number = 0
     Returns 3 * dimensionality + 2 * (dimensionality -1)
  5. when none of these values are 0
     Returns  0.5 * avg-fill-value-digit-number * dimensionality +
              0.5 * avg-valid-range-digit-number * dimensionality +
              2 * (dimensionality -1)"
  [variable params total-estimate]
  (let [dimensionality (get-dimensionality variable)
        avg-fill-value-digit-number (get-avg-fill-value-digit-number variable)
        avg-valid-range-digit-number (get-avg-valid-range-digit-number variable)]
    (log/info (format (str "request-id: %s variable-id: %s total-estimate: %s "
                           "dimensionality: %s avg-fill-value-digit-number: %s "
                           "avg-valid-range-digit-number: %s")
                      (:request-id params) (get-in variable [:meta :concept-id])
                      total-estimate dimensionality avg-fill-value-digit-number
                      avg-valid-range-digit-number))
    (cond 
      (<= dimensionality 0) 0
      (and (= 0 avg-fill-value-digit-number) 
           (not= 0 avg-valid-range-digit-number)) (+ (* avg-valid-range-digit-number dimensionality)
                                                     (* 2 (dec dimensionality)))
      (and (not= 0 avg-fill-value-digit-number) 
           (= 0 avg-valid-range-digit-number)) (+ (* avg-fill-value-digit-number dimensionality)
                                                  (* 2 (dec dimensionality))) 
      (and (= 0 avg-fill-value-digit-number)
           (= 0 avg-valid-range-digit-number)) (+ (* 3 dimensionality)
                                                  (* 2 (dec dimensionality)))
      :else (+ (* 0.5 avg-fill-value-digit-number dimensionality)
               (* 0.5 avg-valid-range-digit-number dimensionality)
               (* 2 (dec dimensionality))))))
