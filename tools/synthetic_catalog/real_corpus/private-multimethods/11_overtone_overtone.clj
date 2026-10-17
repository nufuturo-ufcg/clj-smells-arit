(ns overtone.speech
  (:use
    [overtone.sc.sample :only (sample)]
    [overtone.config.store :only (OVERTONE-DIRS)]
    [clojure.java.shell :only (sh)]
    [overtone.helpers.system :only (get-os)]
    [overtone.helpers.lib :only (uuid)]))

(def ^:private SPEECH-DIR (:speech OVERTONE-DIRS))

(def ^:private VOICES
  {:agnes "Agnes"
   :kathy "Kathy"
   :princess "Princess"
   :vicki "Vicki"
   :victoria "Victoria"

   :bruce "Bruce"
   :fred "Fred"
   :junior "Junior"
   :ralph "Ralph"
   :alex "Alex" ; probably most realistic sounding

   :albert "Albert"
   :zarvox "Zarvox"
   :trinoids "Trinoids"
   :whisper "Whisper"

   :bahh "Bahh"
   :boing "Boing"
   :bubbles "Bubbles"
   :bad-news "Bad News"
   :good-news "Good News"
   :deranged "Deranged"
   :hysterical "Hysterical"

   :bells "Bells"
   :cellos "Cellos"
   :pipe-organ "Pipe Organ"        ; awesome ;-)
   })

(defmulti ^:private say (fn [& args] (get-os)))

(defmethod ^:private say :mac
  [text {:keys [voice]
         :or {voice :vicki}
         :as options}]
  (let [voice (get VOICES voice)
        out-file (str SPEECH-DIR "/" (uuid) ".aiff")]
    (sh "say" "-v" voice "-o" out-file text)
    out-file))
