package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reporter"
	"github.com/thlaurentino/arit/internal/rules"

	_ "github.com/thlaurentino/arit/internal/rules/clojurespecific"
	_ "github.com/thlaurentino/arit/internal/rules/functional"
)

var rootCmd = &cobra.Command{
	Use:   "arit [file-or-dir...]",
	Short: "Arit is a static analyzer for Clojure code.",
	Long: `Arit - Static Analysis for Clojure Code

###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############

Arit analyzes Clojure files for potential issues,
style violations, and opportunities for improvement.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		var startTime time.Time
		if timingFlag {
			startTime = time.Now()
		}
		if !quietFlag {
			fmt.Fprint(os.Stderr, `
###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############

Arit - Static Analysis for Clojure Code

`)
		}

		filesToAnalyze := []string{}

		for _, arg := range args {
			fileInfo, err := os.Stat(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accessing argument %q: %v\n", arg, err)
				continue
			}

			if fileInfo.IsDir() {
				cljFiles, err := findClojureFiles(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding Clojure files in directory %q: %v\n", arg, err)
					continue
				}
				filesToAnalyze = append(filesToAnalyze, cljFiles...)
			} else {
				ext := strings.ToLower(filepath.Ext(arg))
				if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
					filesToAnalyze = append(filesToAnalyze, arg)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Skipping non-Clojure file %q\n", arg)
				}
			}
		}

		if len(filesToAnalyze) == 0 {
			fmt.Fprintln(os.Stderr, "No Clojure files found to analyze.")
			return nil
		}

		sort.Strings(filesToAnalyze)

		configDir := resolveConfigDir(filesToAnalyze)
		cfg, err := config.LoadConfig(configDir)
		if err != nil {
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: Error loading .arit.yaml config from %s: %v. Using default settings.\n", configDir, err)
			}
			cfg = &config.Config{
				EnabledRules: make(map[string]bool),
				RuleConfig:   make(map[string]config.RuleSettings),
			}
		}

		filesToAnalyze = filterTestFiles(filesToAnalyze, cfg)
		outputFormat := reporter.ReportFormat(formatFlag)

		// Delegate heavy execution to runner.go
		allFindings := runAnalysisPipeline(filesToAnalyze, cfg)

		if !quietFlag && outputFormat != reporter.FormatSummary {
			switch outputFormat {
			case reporter.FormatJSON:
				fmt.Fprintf(os.Stderr, "Report generated in JSON format.\n")
			case reporter.FormatJSONSnippet, "json-extended", "jsonsnippet", "json_snippet":
				fmt.Fprintf(os.Stderr, "Report generated in JSON format with code snippets.\n")
			case reporter.FormatHTML:
				fmt.Fprintf(os.Stderr, "Report generated in HTML format.\n")
			case reporter.FormatMarkdown:
				fmt.Fprintf(os.Stderr, "Report generated in Markdown format.\n")
			case reporter.FormatText:
				fmt.Fprintf(os.Stderr, "Report generated in text format.\n")
			default:
				fmt.Fprintf(os.Stderr, "Report generated in %s format.\n", outputFormat)
			}
		}

		if countFindingFlag {
			fmt.Println(len(allFindings))
			return nil
		}

		rep := reporter.NewReporter(outputFormat)
		if rep == nil {
			return fmt.Errorf("unsupported report format: %s", outputFormat)
		}

		var outputWriter io.Writer = os.Stdout
		err = rep.Report(allFindings, outputWriter)
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		if timingFlag {
			duration := time.Since(startTime)
			fmt.Fprintf(os.Stderr, "\nExecution time: %.2fs\n", duration.Seconds())
		}

		return nil
	},
}

var (
	formatFlag       string
	verboseFlag      bool
	timingFlag       bool
	quietFlag        bool
	countFindingFlag bool

	// Advanced Semantic Features
	expCrossNsFlag        bool
	expTypeInferenceFlag  bool
	expAsyncCfgFlag       bool
	expMacroExpansionFlag bool
	inlineSuppressionFlag bool
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "summary", "Output format (summary, text, json, json-snippet, html, markdown, csv, sarif)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&timingFlag, "timing", "t", false, "Show execution time")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress banner and progress output")
	rootCmd.PersistentFlags().BoolVar(&countFindingFlag, "count-finding", false, "Count the total number of findings")

	rootCmd.PersistentFlags().BoolVar(&expCrossNsFlag, "experimental-cross-ns", false, "Enable 2-pass cross-namespace resolution (Experimental)")
	rootCmd.PersistentFlags().BoolVar(&expTypeInferenceFlag, "experimental-types", false, "Enable static type inference and metadata propagation (Experimental)")
	rootCmd.PersistentFlags().BoolVar(&expAsyncCfgFlag, "experimental-async-cfg", false, "Enable core.async channel flow analysis (Experimental)")
	rootCmd.PersistentFlags().BoolVar(&expMacroExpansionFlag, "experimental-macro-expansion", false, "Enable internal macro micro-expansion (Experimental)")
	rootCmd.PersistentFlags().BoolVar(&inlineSuppressionFlag, "inline-suppression", true, "Enable parsing of inline comment directives like ; arit:disable-next-line")
}

var _ = rules.Rule{}
