(ns datahike.reference
  "Cross-database references over datahike's distributed index space.

   Datahike deployments legitimately span MANY databases — per-domain,
   per-tenant, per-permission-boundary — synchronized across peers by
   their store `:id` (a UUID; mandatory in every store config). A
   reference that crosses a database boundary is a triple:

     (db-id, selector, temporal)

   - db-id     the target store's `:id` — the same logical database on
               any peer that replicates it.
   - selector  a LOOKUP REF `[attr value]` on a `:db/unique` attribute —
               datahike's native value-level entity addressing — or a bare
               ENTITY ID (represented as `[:db/id eid]`). An eid is the
               cheapest and most general pointer: a direct EAVT seek, no
               AVET, no schema, works for entities with no unique attr.
               Eids are stable within a logical database (replicas share
               the index, branches share it copy-on-write, datahike never
               reuses eids) but do NOT survive re-materialization
               (export/import, migration-by-retransaction) — value
               selectors do. Physically-bound vs semantically-robust:
               pick per use.
   - temporal  which version of that database:
                 nil               the live head (a LIVING reference —
                                   follows edits; navigation, mentions)
                 {:tx 536871113}   `as-of` a transaction id
                 {:date #inst …}   `as-of` a point in time
                 {:commit #uuid …} an exact content-addressed commit
                                   (via `commit-as-db`) — the most precise
                                   record reference; supersedes :tx/:date
                 {:branch \"exp\"}   a branch head
               A pinned temporal is a RECORD reference — immutable,
               resolvable forever on stores with `:keep-history? true`;
               use it for provenance, citation, audit.

   The URI serialization (for text, hyperlinks, logs, export). The
   temporal is a standard URL query string:

     dh://<db-id>/<attr>/<value>[?<temporal>]

     dh://5d7…c1/entity%2Fuuid/0830…9e                       ; living
     dh://5d7…c1/S.Page%2Ftitle/str:Roadmap?tx=536871113     ; record (as-of tx)
     dh://5d7…c1/entity%2Fuuid/0830…9e?commit=1a2b…          ; record (exact commit)
     dh://5d7…c1/entity%2Fuuid/0830…9e?branch=exploration-2  ; branch head
     dh://5d7…c1/387                                         ; bare entity id

   Attr keywords are URL-encoded whole (`entity%2Fuuid` ≙ :entity/uuid).
   Values carry an optional type tag — `uuid:` `str:` `long:` `kw:` —
   defaulting to uuid-if-it-parses-else-string.

   Related-work correspondence (deliberately rhymed with, not adopted):
   RDF named graph ≙ db-id, resource IRI ≙ selector, predicate ≙ the
   `:dh.ref/type` of a reified reference (below). Export is mechanical.

   ## Reified references

   Inside a database, model an OUTGOING cross-db link as a small entity
   rather than a URI string, so links stay queryable in datalog by
   type/target without string parsing:

     {:dh.ref/db #uuid \"…\" :dh.ref/attr :entity/uuid
      :dh.ref/value \"0830…9e\" :dh.ref/type :derived-from}

   The `:dh.ref/*` attributes live in datahike's system schema (graduated
   like `:db.valid/*` / `:db.secondary/*`), so every store accepts them
   directly — no schema declaration needed. `reference->tx-map` /
   `tx-map->reference` convert. Within-database links should remain plain
   `:db.type/ref` attributes — this namespace is only for crossing stores.

   ## Resolution

   Connection acquisition is deployment-specific (which peer, which
   grants), so `resolve-reference` takes a `connect-fn`:
   `(fn [db-id {:keys [branch]}] conn-or-db-or-nil)`. The temporal is
   applied here via `as-of`. A nil connect result (unknown or ungranted
   store) resolves to nil — a dangling-but-typed reference, which is the
   correct behaviour at a permission boundary."
  (:refer-clojure :exclude [resolve])
  (:require [clojure.string :as str]
            [datahike.api :as d]
            #?(:clj [clojure.edn :as edn] :cljs [cljs.reader :as reader])
            #?(:cljs [goog.crypt.base64 :as gb64]))
  #?(:clj (:import [java.net URLEncoder URLDecoder])))

(defn- url-decode [s]
  #?(:clj  (URLDecoder/decode (str s) "UTF-8")
     :cljs (js/decodeURIComponent (str s))))

(defn- decode-attr [s]
  (keyword (str/join "/" (map url-decode (str/split s #"/")))))
