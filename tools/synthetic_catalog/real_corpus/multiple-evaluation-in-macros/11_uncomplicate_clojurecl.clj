(ns ^{:author "Dragan Djuric"}
    uncomplicate.clojurecl.info
  "Info functions for all OpenCL objects (platforms, devices, etc...).

  The OpenCL standard defines info functions for all cl structures. Typically
  in OpenCL C, you would have a reference to an object representing, for example,
  platform, and then call a dedicated info function, in this case
  [`clGetPlatformInfo`](http://www.jocl.org/doc/org/jocl/CL.html#clGetPlatformInfo-org.jocl.cl_platform_id-int-long-org.jocl.Pointer-long:A-)
  with a parameter param_name that specifies which of the several available
  informations you want about that object. If you need all information, then you
  need to call this function as many times as different kinds of information there is.

  ClojureCL provides many conveniences for obtaining information about cl objects:

  1. **There is a universal, high-level, [[info]] function** that works for all kinds
     of cl objects (platform, context, device, memory, etc.) and displays all available
     information. This function also accepts a keyword argument for returning
     only a specific kind of information, not all information. The information
     will be converted from low-level C enums to a Clojre-friendly format that
     uses keywords, sequences, sets, etc. It will release all additional cl objects
     that it has to use to obtain information. If there is an OpenCL error in obtaining
     the information, which may happen if the driver does not support that kind of
     information, the [ExceptionInfo](http://clojuredocs.org/clojure.core/ex-info)
     will be returned as a result for that particular information, instead of
     raising an exception. This function is useful in when the information
     is going to be displayed to the user.

  2. For each information kind, there is a dedicated, low-level, function that
     returns the raw, unconverted information. If the result is a cl object that
     needs to be released after use, it is the responsibility of the caller to
     call the [[core/release]] function. If the information is not supported,
     the exception is raised. These functions are convenient in the parts
     of the program where the returned info is used by other parts of the program,
     for example to calculate some parameters for an algorithm.

  3. Some information is not only about the objects, for example program, but
     about the specific use of that object, for example a program build. In that
     case, aditional X-info function is provided, for example [[build-info]].

  Most keywords in the [[info]] function are exactly the same as the corresponding
  low-level function name, except in a few cases where that would produce a clash
  with some other functionality. You can check the available keywords in
  the documentation of appropriate positional methods:
  [[->PlatformInfo]], [[->DeviceInfo]], [[->CommandQueueInfo]], [[->ContextInfo]],
  [[->KernelInfo]], [[->KernelArgInfo]], [[->ProgramInfo]], [[->ProgramBuildinfo]],
  [[->EventInfo]], [[->Profilinginfo]], [[->MemObjectInfo]], etc...

  ###Cheat Sheet

  #### Low-level info functions grouped by resource type:

  * [`cl_platform_id`](http://www.jocl.org/doc/org/jocl/cl_platform_id.html) info:
  [[version]], [[icd-suffix-khr]], [[profile]], [[name-info]], [[vendor]],
  [[extensions]]

  * [`cl_device_id`] (http://www.jocl.org/doc/org/jocl/cl_device_id.html) info:
  [[address-bits]], [[available]], [[built-in-kernels]], [[compiler-available]],
  [[double-fp-config]], [[endian-little]], [[error-correction-support]],
  [[execution-capabilities]], [[global-mem-cache-size]], [[global-=mem-cache-type]],
  [[global-mem-cacheline-size]], [[global-mem-size]],
  [[global-variable-preferred-total-size]], [[image2d-max-height]],
  [[image2d-max-width]], [[image3d-max-depth]], [[image3d-max-height]],
  [[image3d-max-width]], [[image-base-address-alignment]], [[image-max-array-size]],
  [[image-max-array-size]], [[image-max-buffer-size]], [[image-pitch-alignment]],
  [[image-support]], [[linker-available]], [[local-mem-size]], [[local-mem-type]],
  [[max-clock-frequency]], [[max-compute-units]], [[max-constant-args]],
  [[max-constant-buffer-size]], [[max-global-variable-size]], [[max-mem-aloc-size]],
  [[max-on-device-events]], [[max-on-device-queues]], [[max-parameter-size]],
  [[max-pipe-args]], [[max-read-image-args]], [[max-read-write-image-args]],
  [[max-samplers]], [[max-work-group-size]], [[max-work-item-dimensions]],
  [[max-work-item-sizes]], [[max-write-image-args]], [[mem-base-addr-align]],
  [[native-vector-width-char]], [[native-vector-width-short]],
  [[native-vector-width-int]], [[native-vector-width-long]],
  [[native-vector-width-float]], [[native-vector-width-double]],
  [[native-vector-width-half]], [[opencl-c-version]], [[parent-device]],
  [[partition-affinity-domain]], [[partition-max-sub-devices]],
  [[partition-properties]], [[partition-type]],[[pipe-max-active-reservations]],
  [[pipe-max-packet-size]], [[platform]], [[preferred-global-atomic-alignment]],
  [[preferred-interop-user-sync]], [[preferred-local-atomic-alignment]],
  [[preferred-platform-atomic-alignment]], [[preferred-vector-width-char]],
  [[preferred-vector-width-short]], [[preferred-vector-width-int]],
  [[preferred-vector-width-long]], [[preferred-vector-width-float]],
  [[preferred-vector-width-double]], [[preferred-vector-width-half]],
  [[printf-buffer-size]], [[profiling-timer-resolution]], [[queue-on-device-max-size]],
  [[queue-on-device-properties]], [[queue-on-host-properties]],
  [[single-fp-config]], [[spir-versions]], [[svm-capabilities]],
  [[device-type]], [[vendor-id]], [[device-version]],
  [[driver-version]], [[extensions]], [[name-info]], [[profile]], [[vendor]],
  [[reference-count]]

  * [`cl_context`] (http://www.jocl.org/doc/org/jocl/cl_context.html) info:
  [[num-devices-in-context]], [[devices-in-context]], [[properties]],
  [[reference-count]]

  * [`cl_command_queue`] (http://www.jocl.org/doc/org/jocl/cl_command_queue.html) info:
  [[queue-context]], [[queue-device]], [[queue-size]], [[properties]],
  [[reference-count]]

  * [`cl_event`] (http://www.jocl.org/doc/org/jocl/cl_event.html) info:
  [[event-command-queue]], [[event-context]], [[command-type]], [[execution-status]],
  [[reference-count]]

  * profiling event info: **[[profiling-info]]**,
  [[queued]], [[submit]], [[start]], [[end]]

  * [`cl_kernel`] (http://www.jocl.org/doc/org/jocl/cl_kernel.html) info:
  [[function-name]], [[num-args]], [[kernel-context]], [[kernel-program]],
  [[attributes]], [[reference-count]]

  * kernel argument info: **[[arg-info]]**
  [[arg-address-qualifier]], [[arg-access-qualifier]], [[arg-type-name]],
  [[arg-type-qualifier]], [[arg-name]]

  * [`cl_mem`] (http://www.jocl.org/doc/org/jocl/cl_mem.html) info:
  [[mem-type]], [[flags]], [[mem-size]], [[map-count]], [[mem-context]],
  [[associated-memobject]], [[offset]], [[uses-svm-pointer]], [[reference-count]]

  * [`cl_program`] (http://www.jocl.org/doc/org/jocl/cl_program.html) info:
  [[program-context]], [[program-num-devices]], [[program-devices]],
  [[program-source]], [[binary-sizes]], [[binaries]], [[program-num-kernels]],
  [[kernel-names]], [[reference-count]]

  * program build info: **[[build-info]]**,
  [[build-status]], [[build-options]], [[build-log]], [[binary-type]],
  [[global-variable-total-size]]

  #### Hihg-level info and keywords (in a few cases different than low-level function names)

  [[->PlatformInfo]], [[->DeviceInfo]], [[->CommandQueueInfo]], [[->ContextInfo]],
  [[->KernelInfo]], [[->KernelArgInfo]], [[->ProgramInfo]], [[->ProgramBuildinfo]],
  [[->EventInfo]], [[->Profilinginfo]], [[->MemObjectInfo]],
  "
  (:require [clojure.string :as str]
            [uncomplicate.commons
             [core :refer [Info info]]
             [utils :refer [unmask unmask1 direct-buffer]]]
            [uncomplicate.fluokitten.core :refer [fmap extract]]
            [uncomplicate.clojurecl.internal
             [constants :refer :all]
             [protocols :refer [wrap]]
             [utils :refer :all]
             [impl :refer :all]])
  (:import [java.nio ByteBuffer IntBuffer LongBuffer]
           [org.jocl CL cl_platform_id  cl_device_id cl_context cl_command_queue
            cl_program cl_kernel cl_sampler cl_event cl_device_partition_property
            cl_mem Sizeof Pointer]
           [uncomplicate.clojurecl.internal.impl CLDevice CLContext CLCommandQueue CLProgram
            CLKernel CLEvent CLBuffer]))

(defmacro ^:private arg-info-string* [kernel arg info]
  `(let [cnt# (long-array 1)
         kernel# (extract ~kernel)
         err# (CL/clGetKernelArgInfo kernel# ~arg ~info 0 nil cnt#)]
     (with-check err#
       (let [res# (byte-array (aget cnt# 0))
             err# (CL/clGetKernelArgInfo kernel# ~arg ~info (alength res#) (Pointer/to res#) nil)]
         (with-check err#
           (String. res# 0 (max 0 (dec (alength res#)))))))))
