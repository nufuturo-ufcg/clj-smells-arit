(ns ^{:skip-wiki true}
  gloss.core.formats
  (:require
    [clj-commons.byte-streams :as bs]
    [gloss.data.bytes.core :refer :all])
  (:import
    [java.nio
     Buffer
     ByteBuffer
     CharBuffer]))
