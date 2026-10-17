(ns beat-link-trigger.track-loader
  "Provides the user interface for exploring the menu trees of the
  available media databases, whether obtained from dbserver or Crate
  Digger (either over the network or from a locally mounted media
  filesystem). For tracks obtained from CDJs, we provide an interface
  for loading tracks into players. When working with the local
  filesytem we provide a simple track selection interface so they can
  be added to show windows."
  (:require [clojure.java.io :as io]
            [clojure.string :as str]
            [beat-link-trigger.menus :as menus]
            [beat-link-trigger.tree-node]
            [beat-link-trigger.util :as util]
            [seesaw.core :as seesaw]
            [seesaw.mig :as mig]
            [seesaw.chooser :as chooser]
            [taoensso.timbre :as timbre])
  (:import [beat_link_trigger.tree_node IMenuEntry ISearchEntry]
           [beat_link_trigger.util PlayerChoice]
           [java.awt.event WindowEvent]
           [java.io File]
           [java.util.concurrent.atomic AtomicInteger]
           [javax.swing JButton JComboBox JDialog JFrame JTree]
           [javax.swing.tree DefaultMutableTreeNode DefaultTreeModel TreeNode TreePath]
           [org.deepsymmetry.beatlink CdjStatus CdjStatus$TrackSourceSlot CdjStatus$TrackType
            DeviceAnnouncement DeviceAnnouncementListener DeviceFinder DeviceUpdateListener
            LifecycleListener VirtualCdj]
           [org.deepsymmetry.beatlink.data MenuLoader MetadataFinder MountListener SlotReference]
           [org.deepsymmetry.beatlink.dbserver Message Message$MenuItemType NumberField StringField]
           [org.deepsymmetry.cratedigger Archivist Archivist$ArchiveListener Database Database$PlaylistFolderEntry]
           [org.deepsymmetry.cratedigger.pdb RekordboxPdb$TrackRow RekordboxPdb$AlbumRow
            RekordboxPdb$ArtistRow RekordboxPdb$GenreRow]))

(defonce ^{:private true
           :doc "Holds the frame allowing the user to pick a track and
  tell a player to load it."} loader-window
  (atom nil))

(defn- empty-node
  "Creates an inert child to explain that its parent is empty. Unless a
  custom label is given, uses [Empty]."
  ([]
   (empty-node "[Empty]"))
  ([label]
   (DefaultMutableTreeNode.
    (proxy [Object IMenuEntry] []
      (toString [] label)
      (getId [] (int 0))
      (getSlot [] nil)
      (getTrackType [] nil)
      (loadChildren [_]))
    false)))

(defn- unloaded?
  "Checks whether a node still needs to be loaded (does not yet have its children)."
  [^DefaultMutableTreeNode node]
  (zero? (.getChildCount node)))

(defn file-track-node
  "Creates a node that represents a track available in an exported
  rekordbox database file. Optionally has an associated slot if it is
  being used with Crate Digger to load tracks from one player onto
  another. If `show-artist` is `true` the artist name will be appended
  to the track name (for disambiguation at the top-level track list)."
  [^Database database ^RekordboxPdb$TrackRow track ^SlotReference slot show-artist]
  (DefaultMutableTreeNode.
   (proxy [Object IMenuEntry] []
     (toString []
       (let [artist-name (when-let [^RekordboxPdb$ArtistRow artist
                                    (when show-artist (.get (.artistIndex database) (.artistId track)))]
                           (Database/getText (.name artist)))]
         (str (Database/getText (.title track))
              (when-not (str/blank? artist-name)
                (str "—" artist-name)))))
     (getId [] (int (.id track)))
     (getSlot [] slot)
     (getTrackType [] CdjStatus$TrackType/REKORDBOX)
     (loadChildren [_]))
   false))

(defn- mark-if-still-empty
  "If, after loading a node, there are still no elements in it, create a
  marker node to make it clear that it is actually empty, and not just
  still loading."
  [^DefaultMutableTreeNode node]
  (when (unloaded? node)
    (.add node (empty-node))))

(defn file-genre-node
  "Creates a node that represents a genre available in an exported
  rekordbox database file. Optionally has an associated slot if it is
  being used with Crate Digger to load tracks from one player onto
  another."
  [^Database database ^RekordboxPdb$GenreRow genre ^SlotReference slot]
  (DefaultMutableTreeNode.
   (proxy [Object IMenuEntry] []
     (toString [] (Database/getText (.name genre)))
     (getId [] (int 0))
     (getSlot [] slot)
     (getTrackType [] nil)
     (loadChildren [^DefaultMutableTreeNode node]
       (when (unloaded? node)
         (doseq [track-id (.. database trackGenreIndex (get (.id genre)))]
           (.add node (file-track-node database (.get (.trackIndex database) track-id) slot true)))
         (mark-if-still-empty node))))
   true))

(defn- file-genres-node
  "Creates a node that contains all the genres available in an exported
  rekordbox database file. Optionally has an associated slot if it is
  being used with Crate Digger to load tracks from one player onto
  another."
  [^Database database ^SlotReference slot]
  (DefaultMutableTreeNode.
   (proxy [Object IMenuEntry] []
     (toString [] "Genres")
     (getId [] (int 0))
     (getSlot [] slot)
     (getTrackType [] nil)
     (loadChildren [^DefaultMutableTreeNode node]
       (when (unloaded? node)
         (doseq [name-genres (.. database genreNameIndex values)]
           (doseq [genre-id name-genres]
             (.add node (file-genre-node database (.get (.genreIndex database) genre-id) slot))))
         (mark-if-still-empty node))))
   true))
