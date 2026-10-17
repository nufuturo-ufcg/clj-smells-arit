(ns puppetlabs.puppetdb.cli.generate
  "# Data Generation utility

   This command-line tool can generate a base sampling of catalog, fact and
   report files suitable for consumption by the PuppetDB benchmark utility.

   Note that it is only necessary to generate a small set of initial sample
   data since benchmark will permute per node differences. So even if you want
   to benchmark 1000 nodes, you don't need to generate initial
   catalog/fact/report json for 1000 nodes.

   If you want a representative sample with big differences between catalogs,
   you will need to run the tool multiple times. For example, if you want a set
   of 5 large catalogs and 10 small ones, you will need to run the tool twice
   with the desired parameters to create the two different sets.

   ## Flag Notes

   ### Catalogs

   #### Resource Counts

   The num-resources flag is total and includes num-classes. So if you set
   --num-resources to 100 and --num-classes to 30, you will get a catalog with a
   hundred resources, thirty of which are classes.

   #### Edges

   A containment edge is always generated between the main stage and each
   class. And non-class resources get a containment edge to a random class. So
   there will always be a base set of containment edges equal to the resource
   count. The --additional-edge-percent governs how many non-containment edges
   are added on top of that to simulate some further catalog structure. There is
   no guarantee of relationship depth (as far as, for example Stage(main) ->
   Class(foo) -> Class(bar) -> Resource(biff)), but it does ensure some edges
   between classes, as well as between class and non-class resources.

   #### Large Resource Parameter Blobs

   The --blob-count and --blob-size parameters control inclusion of large
   text blobs in catalog resources. By default one ~100kb blob is
   added per catalog.

   Set --blob-count to 0 to exclude blobs altogether.

   ### Facts

   #### Baseline Facts

   Each fact set begins with a set of baseline facts from:
   [baseline-agent-node.json](./resources/puppetlabs/puppetdb/generate/samples/facts/baseline-agent-node.json).

   These provide some consistency for a common set of baseline fact paths
   present on any puppet node. The generator then mutates half of the values to
   provide variety.

   #### Fact Counts

   The --num-facts parameter controls the number of facts to generate per host.

   There are 376 leaf facts in the baseline file. Setting num-facts less than
   this will remove baseline facts to approach the requested number of facts.
   (Empty maps and arrays are not removed from the factset, so it will never
   pair down to zero.) Setting num-facts to a larger number will add facts of
   random depth based on --max-fact-depth until the requested count is reached.

   #### Total Facts Size

   The --total-fact-size parameter controls the total weight of the fact values
   map in kB. Weight is added after count is reached. So if the weight of the
   adjusted baseline facts already exceeds the total-fact-size, nothing more is
   done. No attempt is made to pair facts back down the requested size, as this
   would likely require removing facts.

   #### Max Fact Depth

   The --max-fact-depth parameter is the maximum nested depth a fact added to
   the baseline facts may reach. For example a max depth of 5, would mean that
   an added fact would at most be a nest of four maps:

     {foo: {bar: {baz: {biff: boz}}}}

   Since depth is picked randomly for each additional fact, this does not
   guarantee facts of a given depth. Nor does it directly affect the average
   depth of facts in the generated factset, although the larger the
   max-fact-depth and num-facts, the more likely that the average depth will
   drift higher.

   #### Package Inventory

   The --num-packages parameter sets the number of packages to generate for the
   factset's package_inventory array. Set to 0 to exclude.

   ### Reports

   #### Reports per Catalog

   The --num-reports flag governs the number of reports to generate per
   generated catalog.  Since one catalog is generated per host, this means you
   will end up with num-hosts * num-reports reports.

   #### Variation in Reports

   A report details change, or lack there of, during enforcement of the puppet
   catalog on the host. Since the benchmark tool currently chooses randomly from the
   given report files, a simple mechanism for determining the likelihood of
   receiving a report of a particular size (with lots of changes, few changes or
   no changes) is to produce multiple reports of each type per host to generate
   a weighted average. (If there are 10 reports, 2 are large and 8 are small,
   then it's 80% likely any given report submission submitted by benchmark will
   be of the small variety...)

   The knobs to control this with the generate tool are:

   * --num-reports, to determine the base number of reports to generate per catalog
   * --high-change-reports-percent, percentage of that base to generate as
     reports with a high number of change events, as determined by:
   * --high-change-resource-percent, percentage of resources in a high change
     report that will experience events (changes)
   * --low-change-reports-percent, percentage of the base reports to generate
     as reports with a low number of change events as determined by:
   * --low-change-resource-percent, percentage of resources in a low change
     report that will experience events (changes)

   The left over percentage of reports will be no change reports (generally the
   most common) indicating the report run was steady-state with no changes.

   By default, with a num-reports of 20, a high change percent of 5% and a low
   change percent of 20%, you will get 1 high change, 4 low change and 15
   unchanged reports per host.

   #### Unchanged Resources

   In Puppet 8, by default, the agent no longer includes unchanged resources in
   the report, reducing its size.

   The generate tool also does this by default, but you can set
   --no-exclude-unchanged-resources to instead include unchanged resources in
   every report (for default Puppet 7 behavior, for example).

   #### Logs

   In addition to a few boilerplate log lines, random logs are generated for
   each change event in the report. However other factors, such as pluginsync,
   puppet runs with debug lines and additional logging in modules can increase
   log output (quite dramatically in the case of debug output from the agent).

   To simulate this, you can set --num-additional-logs to include in a report.
   And you can set --percent-add-report-logs to indicate what percentage of
   reports have this additional number of logs included.

   ### Random Distribution

   The default generation produces relatively uniform structures.

   * for catalogs it generates equal resource and edge counts and similar byte
     counts.
   * for factsets it generates equal fact counts and similar byte counts.

   Example:

      jpartlow@jpartlow-dev-2204:~/work/src/puppetdb$ lein run generate --verbose --output-dir generate-test
      ...
      :catalogs: 5

      |     :certname | :resource-count | :resource-weight | :min-resource | :mean-resource | :max-resource | :edge-count | :edge-weight | :catalog-weight |
      |---------------+-----------------+------------------+---------------+----------------+---------------+-------------+--------------+-----------------|
      | host-sarasu-0 |             101 |           137117 |            90 |           1357 |        110246 |         150 |        16831 |          154248 |
      | host-lukoxo-1 |             101 |           132639 |            98 |           1313 |        104921 |         150 |        16565 |          149504 |
      | host-dykivy-2 |             101 |           120898 |           109 |           1197 |         94013 |         150 |        16909 |          138107 |
      | host-talyla-3 |             101 |           110328 |           128 |           1092 |         82999 |         150 |        16833 |          127461 |
      | host-foropy-4 |             101 |           136271 |           106 |           1349 |        109811 |         150 |        16980 |          153551 |

      :facts: 5

      |     :certname | :fact-count | :avg-depth | :max-depth | :fact-weight | :total-weight |
      |---------------+-------------+------------+------------+--------------+---------------|
      | host-sarasu-0 |         400 |       2.77 |          7 |        10000 |         10118 |
      | host-lukoxo-1 |         400 |        2.8 |          7 |        10000 |         10118 |
      | host-dykivy-2 |         400 |     2.7625 |          7 |        10000 |         10118 |
      | host-talyla-3 |         400 |     2.7825 |          7 |        10000 |         10118 |
      | host-foropy-4 |         400 |     2.7925 |          7 |        10000 |         10118 |
      ...

   This mode is best used when generating several different sample sets with
   distinct weights and counts to provide (when combined) an overall sample set
   for benchmark that includes some fixed number of fairly well described
   catalog, fact and report examples.

   By setting --random-distribution to true, you can instead generate a more random
   sample set, where the exact parameter values used per host will be picked
   from a normal curve based on the set value as mean.

   * for catalogs, this will effect the class, resource, edge and total blob counts

   Blobs will be distributed randomly through the set, so if you
   set --blob-count to 2 over --hosts 10, on averge there will be two per
   catalog, but some may have none, others four, etc...

   * for facts, this will effect the fact and package counts, the total weight and the max fact depth.

   This has no effect on generated reports at the moment.

   Example:

      jpartlow@jpartlow-dev-2204:~/work/src/puppetdb$ lein run generate --verbose --random-distribution
      :catalogs: 5

      |     :certname | :resource-count | :resource-weight | :min-resource | :mean-resource | :max-resource | :edge-count | :edge-weight | :catalog-weight |
      |---------------+-----------------+------------------+---------------+----------------+---------------+-------------+--------------+-----------------|
      | host-cevani-0 |             122 |            33831 |            93 |            277 |           441 |         193 |        22044 |           56175 |
      | host-firilo-1 |              91 |           115091 |           119 |           1264 |         91478 |         130 |        14466 |          129857 |
      | host-gujudi-2 |             129 |            36080 |           133 |            279 |           465 |         180 |        20230 |           56610 |
      | host-xegyxy-3 |             106 |           120603 |           136 |           1137 |         92278 |         153 |        17482 |          138385 |
      | host-jaqomi-4 |             107 |           211735 |            87 |           1978 |         98354 |         159 |        17792 |          229827 |

      :facts: 5

      |     :certname | :fact-count | :avg-depth | :max-depth | :fact-weight | :total-weight |
      |---------------+-------------+------------+------------+--------------+---------------|
      | host-cevani-0 |         533 |  3.4690433 |          9 |        25339 |         25457 |
      | host-firilo-1 |         355 |  2.7464788 |          7 |        13951 |         14069 |
      | host-gujudi-2 |         380 |       2.75 |          8 |        16111 |         16229 |
      | host-xegyxy-3 |         360 |  2.7305555 |          7 |         5962 |          6080 |
      | host-jaqomi-4 |         269 |  2.7695167 |          7 |        16984 |         17102 |
      ..."
  (:require
    [clojure.data.generators :as dgen]
    [clojure.java.io :as io]
    [clojure.pprint :as pp]
    [clojure.set :as cset]
    [clojure.string :as string]
    [clojure.tools.namespace.dependency :as dep]
    [puppetlabs.i18n.core :refer [trs]]
    [puppetlabs.kitchensink.core :as kitchensink]
    [puppetlabs.puppetdb.cheshire :as json]
    [puppetlabs.puppetdb.cli.util :refer [exit run-cli-cmd]]
    [puppetlabs.puppetdb.export :as export]
    [puppetlabs.puppetdb.facts :as facts]
    [puppetlabs.puppetdb.nio :refer [get-path]]
    [puppetlabs.puppetdb.random :as rnd]
    [puppetlabs.puppetdb.time :as time :refer [now]]
    [puppetlabs.puppetdb.utils :as utils])
  (:import
    [java.nio.file.attribute FileAttribute]
    [java.nio.file Files]
    [org.apache.commons.lang3 RandomStringUtils]))

(def environment "production")

(defn generate-log
  "Create one log entry for a report's log array."
  ([level message] (generate-log {:level level :message message}))
  ([{:keys [file line level message source tags]
     :or {level (rand-nth ["info" "notice"])
          message (rnd/random-sentence-ish)
          tags #{level}
          source "Puppet"}}]
   (let [final-tags (cset/union (set tags) #{level})]
     {:file file
      :line line
      :level level
      :message message
      :source source
      :tags final-tags
      :time (now)})))

(defn generate-report-logs
  "Generate log of changes for a report based on a set of changed resources
   and some boilerplate."
  [catalog changed-resources]
  (let [headers (if (> (count changed-resources) 0)
                  (map #(generate-log "info" %)
                       [(format "Using environment '%s'" (:environment catalog))
                        "Retrieving pluginfacts"
                        "Retrieving plugin"
                        "Loading facts"
                        (format "Applying configuration version '%s'" (:version catalog))])
                  [])
        footers [(generate-log "notice" (format "Applied catalog in %s.%s seconds"
                                                (rand-int 100) (rand-int 100)))]
        changes (map generate-log changed-resources)]
    (-> (concat headers changes)
        (concat footers))))
