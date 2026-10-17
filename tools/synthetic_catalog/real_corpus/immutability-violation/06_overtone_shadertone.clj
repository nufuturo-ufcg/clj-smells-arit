(ns #^{:author "Roger Allen"
       :doc "Shadertoy-like core library."}
  shadertone.shader
  (:require [watchtower.core :as watcher]
            clojure.string)
  (:import (java.awt.image BufferedImage DataBuffer DataBufferByte WritableRaster)
           (java.io File FileInputStream)
           (java.nio IntBuffer ByteBuffer FloatBuffer ByteOrder)
           (java.util Calendar)
           (javax.imageio ImageIO)
           (java.lang.reflect Field)
           (org.lwjgl BufferUtils)
           (org.lwjgl.input Mouse)
           (org.lwjgl.opengl ContextAttribs Display DisplayMode
                             GL11 GL12 GL13 GL15 GL20
                             PixelFormat)))

(defonce default-state-values
  {:active              :no  ;; :yes/:stopping/:no
   :width               0
   :height              0
   :title               ""
   :display-sync-hz     60
   :start-time          0
   :last-time           0
   ;; mouse
   :mouse-clicked       false
   :mouse-pos-x         0
   :mouse-pos-y         0
   :mouse-ori-x         0
   :mouse-ori-y         0
   ;; geom ids
   :vbo-id              0
   :vertices-count      0
   ;; shader program
   :shader-good         true ;; false in error condition
   :shader-filename     nil
   :shader-str-atom     (atom nil)
   :shader-str          ""
   :vs-id               0
   :fs-id               0
   :pgm-id              0
   ;; shader uniforms
   :i-resolution-loc    0
   :i-global-time-loc   0
   :i-channel-time-loc  0
   :i-mouse-loc         0
   :i-channel-loc       [0 0 0 0]
   :i-channel-res-loc   0
   :i-date-loc          0
   :channel-time-buffer (-> (BufferUtils/createFloatBuffer 4)
                            (.put (float-array
                                   [0.0 0.0 0.0 0.0]))
                            (.flip))
   :channel-res-buffer (-> (BufferUtils/createFloatBuffer (* 3 4))
                            (.put (float-array
                                   [0.0 0.0 0.0
                                    0.0 0.0 0.0
                                    0.0 0.0 0.0
                                    0.0 0.0 0.0]))
                            (.flip))
   ;; textures
   :tex-filenames       []
   :tex-ids             []
   :tex-types           [] ; :cubemap, :previous-frame
   ;; a user draw function
   :user-fn             nil
   ;; pixel read
   :pixel-read-enable   false
   :pixel-read-pos-x    0
   :pixel-read-pos-y    0
   :pixel-read-data      (-> (BufferUtils/createByteBuffer 3)
                            (.put (byte-array (map byte [0 0 0])))
                            (.flip))
   })

(defonce the-window-state (atom default-state-values))

(defonce throw-on-gl-error (atom true))

(defonce pixel-value (atom [0.0 0.0 0.0]))

(defn- get-fields [#^Class static-class]
  (. static-class getFields))

(defn- gl-enum-name
  "Takes the numeric value of a gl constant (i.e. GL_LINEAR), and gives the name"
  [enum-value]
  (if (= 0 enum-value)
    "NONE"
    (.getName #^Field (some
                       #(if (= enum-value (.get #^Field % nil)) % nil)
                       (mapcat get-fields [GL11 GL12 GL13 GL15 GL20])))))

(defn- except-gl-errors
  [msg]
  (let [error (GL11/glGetError)
        error-string (str "OpenGL Error(" error "):"
                          (gl-enum-name error) ": " msg)]
    (if (and (not (zero? error)) @throw-on-gl-error)
      (throw (Exception. error-string)))))

(defn- buffer-swizzle-0123-1230
  "given a ARGB pixel array, swizzle it to be RGBA.  Or, ABGR to BGRA"
  ^bytes [^bytes data] ;; Wow!  That ^bytes changes this from 10s for a 256x256 tex to instantaneous.
  (dotimes [i (/ (alength data) 4)]
    (let [i0 (* i 4)
          i1 (inc i0)
          i2 (inc i1)
          i3 (inc i2)
          tmp (aget data i0)]
      (aset data i0 (aget data i1))
      (aset data i1 (aget data i2))
      (aset data i2 (aget data i3))
      (aset data i3 tmp)))
  data)

(defn- get-pixel-value
  [^ByteBuffer rgb-bytes]
  (let [rf (/ (float (int (bit-and 0xFF (.get rgb-bytes 0)))) 255.0)
        gf (/ (float (int (bit-and 0xFF (.get rgb-bytes 1)))) 255.0)
        bf (/ (float (int (bit-and 0xFF (.get rgb-bytes 2)))) 255.0)]
    [rf gf bf]))

(defn- draw
  [locals]
  (let [{:keys [width height i-resolution-loc
                start-time last-time i-global-time-loc
                i-date-loc
                pgm-id vbo-id
                vertices-count
                i-mouse-loc
                mouse-pos-x mouse-pos-y
                mouse-ori-x mouse-ori-y
                i-channel-time-loc i-channel-loc
                i-channel-res-loc
                channel-time-buffer channel-res-buffer
                old-pgm-id old-fs-id
                tex-ids tex-types
                user-fn
                pixel-read-enable
                pixel-read-pos-x pixel-read-pos-y
                pixel-read-data]} @locals
        cur-time    (/ (- last-time start-time) 1000.0)
        _           (.put ^FloatBuffer channel-time-buffer 0 (float cur-time))
        _           (.put ^FloatBuffer channel-time-buffer 1 (float cur-time))
        _           (.put ^FloatBuffer channel-time-buffer 2 (float cur-time))
        _           (.put ^FloatBuffer channel-time-buffer 3 (float cur-time))
        cur-date    (Calendar/getInstance)
        cur-year    (.get cur-date Calendar/YEAR)         ;; four digit year
        cur-month   (.get cur-date Calendar/MONTH)        ;; month 0-11
        cur-day     (.get cur-date Calendar/DAY_OF_MONTH) ;; day 1-31
        cur-seconds (+ (* (.get cur-date Calendar/HOUR_OF_DAY) 60.0 60.0)
                       (* (.get cur-date Calendar/MINUTE) 60.0)
                       (.get cur-date Calendar/SECOND))]

    (except-gl-errors "@ draw before clear")

    (GL11/glClear GL11/GL_COLOR_BUFFER_BIT)

    (when user-fn
      (user-fn :pre-draw pgm-id))

    ;; activate textures
    (dotimes [i (count tex-ids)]
      (when (nth tex-ids i)
        (GL13/glActiveTexture (+ GL13/GL_TEXTURE0 i))
        (cond
         (= :cubemap (nth tex-types i))
         (GL11/glBindTexture GL13/GL_TEXTURE_CUBE_MAP (nth tex-ids i))
         (= :previous-frame (nth tex-types i))
         (GL11/glBindTexture GL11/GL_TEXTURE_2D (nth tex-ids i))
         :default
         (GL11/glBindTexture GL11/GL_TEXTURE_2D (nth tex-ids i)))))

    (except-gl-errors "@ draw after activate textures")

    ;; setup our uniform
    (GL20/glUniform3f i-resolution-loc width height 1.0)
    (GL20/glUniform1f i-global-time-loc cur-time)
    (GL20/glUniform1  ^Integer i-channel-time-loc ^FloatBuffer channel-time-buffer)
    (GL20/glUniform4f i-mouse-loc
                      mouse-pos-x
                      mouse-pos-y
                      mouse-ori-x
                      mouse-ori-y)
    (GL20/glUniform1i (nth i-channel-loc 0) 0)
    (GL20/glUniform1i (nth i-channel-loc 1) 1)
    (GL20/glUniform1i (nth i-channel-loc 2) 2)
    (GL20/glUniform1i (nth i-channel-loc 3) 3)
    (GL20/glUniform3  ^Integer i-channel-res-loc ^FloatBuffer channel-res-buffer)
    (GL20/glUniform4f i-date-loc cur-year cur-month cur-day cur-seconds)
    ;; get vertex array ready
    (GL11/glEnableClientState GL11/GL_VERTEX_ARRAY)
    (GL15/glBindBuffer GL15/GL_ARRAY_BUFFER vbo-id)
    (GL11/glVertexPointer 4 GL11/GL_FLOAT 0 0)

    (except-gl-errors "@ draw prior to DrawArrays")

    ;; Draw the vertices
    (GL11/glDrawArrays GL11/GL_TRIANGLES 0 vertices-count)

    (except-gl-errors "@ draw after DrawArrays")

    ;; Put everything back to default (deselect)
    (GL15/glBindBuffer GL15/GL_ARRAY_BUFFER 0)
    (GL11/glDisableClientState GL11/GL_VERTEX_ARRAY)
    ;; unbind textures
    (dotimes [i (count tex-ids)]
      (when (nth tex-ids i)
        (GL13/glActiveTexture (+ GL13/GL_TEXTURE0 i))
        (GL11/glBindTexture GL13/GL_TEXTURE_CUBE_MAP 0)
        (GL11/glBindTexture GL11/GL_TEXTURE_2D 0)))

    (except-gl-errors "@ draw prior to post-draw")

    (when user-fn
      (user-fn :post-draw pgm-id))

    (except-gl-errors "@ draw after post-draw")

    (GL20/glUseProgram 0)

    ;; copy the rendered image
    (dotimes [i (count tex-ids)]
      (when (= :previous-frame (nth tex-types i))
        (GL11/glBindTexture GL11/GL_TEXTURE_2D (nth tex-ids i))
        (GL11/glCopyTexImage2D GL11/GL_TEXTURE_2D 0 GL11/GL_RGBA8 0 0 width height 0)
        (GL11/glBindTexture GL11/GL_TEXTURE_2D 0)))
    (except-gl-errors "@ draw after copy")

    ;; read a pixel value
    (when pixel-read-enable
      (GL11/glReadPixels ^Integer pixel-read-pos-x ^Integer pixel-read-pos-y
                        1 1
                        GL11/GL_RGB GL11/GL_UNSIGNED_BYTE
                        ^ByteBuffer pixel-read-data)
      (except-gl-errors "@ draw after pixel read")
      (reset! pixel-value (get-pixel-value ^ByteBuffer pixel-read-data)))))

(defn pixel-read-enable!
  "Enable reading a pixel each frame from location x,y.  Be sure x,y
   are valid or things may crash!  Results are available via the
   function (pixel) and via the atom @pixel-value"
  [x y]
  (swap! the-window-state assoc
         :pixel-read-enable true
         :pixel-read-pos-x  x
         :pixel-read-pos-y  y)
  nil)

(defn pixel
  "Return the data that was read from the currently drawn frame at the
  x,y location specified in the (pixel-read-enable! x y) call.  When
  enabled, a [red green blue] vector of floating point [0.0,1.0]
  values is returned.  Otherwise, [0.0 0.0 0.0] is returned."
  []
  (if (:pixel-read-enable @the-window-state)
    (get-pixel-value (:pixel-read-data @the-window-state))
    [0.0 0.0 0.0]))
