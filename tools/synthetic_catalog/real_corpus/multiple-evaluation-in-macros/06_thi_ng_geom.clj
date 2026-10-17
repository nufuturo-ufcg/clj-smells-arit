(ns thi.ng.geom.macros.vector
  (:require
   [clojure.walk :refer :all]
   [thi.ng.dstruct.core :as d]
   [thi.ng.xerror.core :as err]))

(defmacro rewrite-v3-nv-no-let
  [src v default & body]
  (let [a (gensym), b (gensym)]
    `(let [~(with-meta a {:tag "doubles"}) ~src]
       (if (instance? ~'Vec3 ~v)
         (let [~(with-meta b {:tag "doubles"}) (.-buf ~(with-meta v {:tag "Vec3"}))]
           ~@(postwalk-replace
              {'x `(aget ~a 0) 'y `(aget ~a 1) 'z `(aget ~a 2)
               'vx `(aget ~b 0) 'vy `(aget ~b 1) 'vz `(aget ~b 2)}
              body))
         (if (number? ~v)
           (do ~@(postwalk-replace
                  {'x `(aget ~a 0) 'y `(aget ~a 1) 'z `(aget ~a 2) 'vx v 'vy v 'vz v} body))
           (do ~@(postwalk-replace
                  {'x `(aget ~a 0) 'y `(aget ~a 1) 'z `(aget ~a 2)
                   'vx `(nth ~v 0 ~default) 'vy `(nth ~v 1 ~default) 'vz `(nth ~v 2 ~default)}
                  body)))))))

(defmacro v3-op1!
  [op src v]
  `(rewrite-v3-nv-no-let
    ~src ~v 0.0
    (aset ~src 0 (double (~op ~'x ~'vx)))
    (aset ~src 1 (double (~op ~'y ~'vy)))
    (aset ~src 2 (double (~op ~'z ~'vz)))
    (set! ~'_hash nil)))
