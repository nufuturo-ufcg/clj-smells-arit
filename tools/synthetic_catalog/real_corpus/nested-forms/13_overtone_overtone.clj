(ns overtone.examples.notation.mad
  {:clj-kondo/config '{:linters {:unresolved-symbol {:level :off}}}}
  (:use overtone.live
        overtone.inst.synth))

(definst tone [note 60 amp 0.3 dur 0.4]
  (let [snd (sin-osc (midicps note))
        env (env-gen (perc 0.01 dur) :action FREE)]
    (* env snd amp)))

(defn defnote
  [n-sym pitch]
  (intern *ns* n-sym
          {:synth tone
           :amp 0.2
           :pitch pitch
           :dur 0.1
           :data []}))

(defn def-notes
  "Define vars for all notes."
  []
  (doseq [octave (range 8)]
    (doseq [n (range 7)]
      (let [n-char (char (+ 65 n))
            n-sym (symbol (str n-char octave))
            note (octave-note octave (get NOTES (keyword (str n-char))))]
        (defnote n-sym note)
        (when-let [sharp (get NOTES (keyword (str n-char "#")))]
          (defnote (symbol (str n-char "#" octave))
                   (octave-note octave sharp)))
        (when-let [flat (get NOTES (keyword (str n-char "b")))]
          (defnote (symbol (str n-char "b" octave))
                   (octave-note octave flat)))))))
