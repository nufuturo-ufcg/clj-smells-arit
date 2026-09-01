package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/catalog"
)

var (
	catalogManifestFlag string
	catalogJSONFlag     bool
	catalogCrossNSFlag  bool
)

var catalogAuditCmd = &cobra.Command{
	Use:   "catalog-audit [catalog-directory]",
	Short: "Audit the expanded synthetic catalog against its semantic manifest",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath := catalogManifestFlag
		if manifestPath == "" {
			manifestPath = "docs/CATALOG_SEMANTIC_EXPECTATIONS.yaml"
		}
		if !filepath.IsAbs(manifestPath) {
			if absolute, err := filepath.Abs(manifestPath); err == nil {
				manifestPath = absolute
			}
		}
		report, err := catalog.Audit(args[0], manifestPath, catalog.Options{
			ExperimentalCrossNamespace: catalogCrossNSFlag,
		})
		if err != nil {
			return err
		}
		if catalogJSONFlag {
			return json.NewEncoder(os.Stdout).Encode(report)
		}
		fmt.Printf("files=%d findings=%d proven=%d contextual=%d no_findings=%d\n", len(report.Files), report.Findings, report.Proven, report.Contextual, len(report.NoFindingFiles))
		for _, file := range report.NoFindingFiles {
			reason := file.Expectation.NoFindingReason
			if reason == "" {
				reason = "no rule proved a finding; this file remains a valid negative or unsupported semantic case"
			}
			fmt.Printf("NO-FINDING %s: %s\n", file.Path, reason)
		}
		for _, item := range report.ContextualWithoutReason {
			fmt.Printf("CONTEXTUAL-WITHOUT-EVIDENCE %s\n", item)
		}
		for _, violation := range report.Violations {
			fmt.Printf("VIOLATION %s\n", violation)
		}
		if len(report.ContextualWithoutReason) > 0 || len(report.Violations) > 0 {
			return fmt.Errorf("catalog audit failed")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(catalogAuditCmd)
	catalogAuditCmd.Flags().StringVar(&catalogManifestFlag, "manifest", "", "Path to the semantic catalog manifest")
	catalogAuditCmd.Flags().BoolVar(&catalogJSONFlag, "json", false, "Emit the audit report as JSON")
	catalogAuditCmd.Flags().BoolVar(&catalogCrossNSFlag, "experimental-cross-ns", false, "Enable cross-namespace resolution during the audit")
}
