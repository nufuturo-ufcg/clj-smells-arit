(ns overtone.midi.file
  (:import java.io.File
           java.net.URL
           [javax.sound.midi MidiSystem MidiFileFormat Sequence Track
            MetaMessage ShortMessage])
  (:use [overtone.midi :only (midi-msg)]))

(defn- midi-division-type
  [info]
  (case (.getDivisionType info)
    Sequence/PPQ          :ppq
    Sequence/SMPTE_24     :smpte-24fps
    Sequence/SMPTE_25     :smpte-25fps
    Sequence/SMPTE_30DROP :smpte-30drop
    Sequence/SMPTE_30     :smpte-30fps
    :unknown))
