(ns examples.smells.inappropriate-collection-validation)

;; Test cases for InappropriateCollectionRule

(def my-vector [1 2 3 4 5])
(def my-list (list 1 2 3 4 5))
(def my-map {:a 1 :b 2 :c 3})
(def my-set #{1 2 3 4 5})

;; Violations expected
(last my-vector)       ; Inappropriate: last on vector
(nth my-list 2)        ; Inappropriate: nth on list for access

(contains? my-vector :a) ; Inappropriate: contains? on vector with non-numeric key
(contains? my-list :b)   ; Inappropriate: contains? on list with non-numeric key (should use set/map)
(contains? my-list "value") ; Inappropriate: contains? on list with non-numeric key (should use set/map)


;; Valid uses (should NOT be flagged by this specific rule's checks)
(peek my-vector)
(get my-vector 2)
(nth [0 1 2] 1) ; nth on literal vector is fine for small, fixed-size vector access

(contains? my-map :a)
(contains? my-set 3)
(contains? my-vector 2) ; Valid: contains? on vector with numeric key (index check)

(def another-vec (vec (range 100)))
(last another-vec) ; Should flag

(def another-list (apply list (range 100)))
(nth another-list 50) ; Should flag 