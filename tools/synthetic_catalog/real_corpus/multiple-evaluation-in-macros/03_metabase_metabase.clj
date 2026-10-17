(ns metabase.settings.core
  "Settings are a fast and simple way to create a setting that can be set from the admin page. They are saved to the
  application Database, but intelligently cached internally for super-fast lookups.

  Define a new Setting with [[defsetting]] (optionally supplying things like default value, type, or custom getters &
  setters):

    (defsetting mandrill-api-key \"API key for Mandrill\")

  The newly-defined Setting will automatically be made available to the frontend client depending on its [[Visibility]].

  You can also set the value via the corresponding env var, which looks like `MB_MANDRILL_API_KEY`, where the name of
  the Setting is converted to uppercase and dashes to underscores.

  The var created with [[defsetting]] can be used as a getter/setter, or you can use [[get]] and [[set!]]:

    (require '[metabase.settings.models.setting :as setting])

    (setting/get :mandrill-api-key) ; only returns values set explicitly from the Admin Panel
    (mandrill-api-key)              ; returns value set in the Admin Panel, OR value of corresponding env var,
                                    ; OR the default value, if any (in that order)

    (setting/set! :mandrill-api-key \"NEW_KEY\")
    (mandrill-api-key! \"NEW_KEY\")

    (setting/set! :mandrill-api-key nil)
    (mandrill-api-key! nil)

  You can define additional Settings types adding implementations of [[default-tag-for-type]], [[get-value-of-type]],
  and [[set-value-of-type!]].

  [[writable-settings]] and [[user-readable-values-map]] can be used to fetch *all* Admin-writable and
  User-readable Settings, respectively. See their docstrings for more information.

  ### User-local and Database-local Settings

  Starting in 0.42.0, some Settings are allowed to have Database-specific values that override the normal site-wide
  value. Similarly, starting in 0.43.0, some Settings are allowed to have User-specific values. These are similar in
  concept to buffer-local variables in Emacs Lisp.

  When a Setting is allowed to be User or Database local, any values in [[*user-local-values*]] or
  [[*database-local-values*]] for that Setting will be returned preferentially to site-wide values of that Setting.
  [[*user-local-values*]] comes from the `User.settings` column in the application DB, and [[*database-local-values*]]
  comes from the `Database.settings` column. `nil` values in [[*user-local-values*]] and [[*database-local-values*]]
  are ignored, i.e. you cannot 'unset' a site-wide value with a User- or Database-local one.

  Whether or not a Setting can be User- or Database-local is controlled by the `:user-local` and `:database-local`
  options passed to [[defsetting]]. A Setting can only be User-local *or* Database-local, not both; this is enforced
  when the Setting is defined. There are three valid values of these options:

  * `:only` means this Setting can *only* have a User- or Database-local value and cannot have a 'normal' site-wide
  value. It cannot be set via env var. Default values are still allowed for User- and Database-local-only Settings.
  Database-local-only Settings are never returned by [[writable-settings]] or [[user-readable-values-map]] regardless of
  their [[Visibility]].

  * `:allowed` means this Setting can be User- or Database-local and can also have a normal site-wide value; if both
  are specified, the User- or Database-specific value will be returned preferentially when we are in the context of a
  specific User or Database (i.e., [[*user-local-values*]] or [[*database-local-values*]] is bound).

  * `:never` means User- or Database-specific values cannot be set for this Setting. Values in [[*user-local-values*]]
  and [[*database-local-values*]] will be ignored.

  `:never` is the default value of both `:user-local` and `:database-local`; to allow User- or Database-local values,
  the Setting definition must explicitly specify `:only` or `:allowed` for the appropriate option.

  If a User-local setting is written in the context of an API request (i.e., when [[metabase.api.common/*current-user*]]
  is bound), the value will be local to the current user. If it is written outside of an API request, a site-wide
  value will be written. (At the time of this writing, there is not yet a FE-client-friendly way to set Database-local
  values. Just set them manually in the application DB until we figure that out.)

  Custom setter functions do not affect User- or Database-local values; they always set the site-wide value.

  See #14055 and #19399 for more information about and motivation behind User- and Database-local Settings."
  (:refer-clojure :exclude [get])
  (:require
   [metabase.settings.models.setting]
   [metabase.settings.models.setting.cache]
   [metabase.settings.models.setting.multi-setting]
   [metabase.settings.settings]
   [potemkin :as p]))

(defmacro with-database
  "Execute `body` with Database-local Setting values bound to `new-values`."
  [new-db & body]
  `(binding [metabase.settings.models.setting/*database*              (:metabase/toucan-instance (meta ~new-db))
             metabase.settings.models.setting/*database-local-values* (or (:settings ~new-db) {})]
     ~@body))
