package cmd

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/stats_collector"
)

var (
	outputDir string
	rawOutput bool
)

func init() {
	statsCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to save the output CSV files. Defaults to a new directory in the current path.")
	statsCmd.Flags().BoolVar(&rawOutput, "raw", false, "Output raw, non-aggregated stats to a single CSV file for debugging.")
	rootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Collect statistics from a Clojure project",
	Long: `Analyzes a directory of Clojure files to gather statistics about function definitions, 
such as lines of code, parameter count, and nesting depth.

The command aggregates this data and outputs CSV files summarizing the distribution of these metrics.
This is useful for understanding the characteristics of a codebase and for defining
accurate thresholds for linting rules.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := args[0]

		allStats, err := collectStatsFromPath(projectPath)
		if err != nil {
			return fmt.Errorf("error collecting stats: %w", err)
		}

		if rawOutput {
			return writeRawOutput(allStats)
		}
		return writeAggregatedOutput(allStats)
	},
}

func collectStatsFromPath(projectPath string) ([]stats_collector.FunctionStats, error) {
	var allStats []stats_collector.FunctionStats
	err := filepath.Walk(projectPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isClojureFile(info.Name()) {
			tree, err := reader.ParseFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing file %s: %v\n", path, err)
				return nil
			}

			richRoots, _ := reader.BuildRichTree(tree)
			for _, root := range richRoots {
				fileStats := stats_collector.Collect(root)
				allStats = append(allStats, fileStats...)
			}
		}
		return nil
	})
	return allStats, err
}

func isClojureFile(filename string) bool {
	return strings.HasSuffix(filename, ".clj") ||
		strings.HasSuffix(filename, ".cljs") ||
		strings.HasSuffix(filename, ".cljc")
}

func writeRawOutput(stats []stats_collector.FunctionStats) error {
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputDir, "raw_stats.csv")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create raw stats file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"function_name", "metric_type", "metric_value"}); err != nil {
		return fmt.Errorf("failed to write raw stats header: %w", err)
	}

	for _, s := range stats {
		if err := writer.Write([]string{s.FunctionName, "lines_of_code", strconv.Itoa(s.LinesOfCode)}); err != nil {
			return err
		}
		if err := writer.Write([]string{s.FunctionName, "parameter_count", strconv.Itoa(s.ParameterCount)}); err != nil {
			return err
		}
		if err := writer.Write([]string{s.FunctionName, "max_nesting_depth", strconv.Itoa(s.MaxNestingDepth)}); err != nil {
			return err
		}
		if err := writer.Write([]string{s.FunctionName, "max_message_chain", strconv.Itoa(s.MaxMessageChain)}); err != nil {
			return err
		}
		if err := writer.Write([]string{s.FunctionName, "max_consecutive_primitives", strconv.Itoa(s.MaxConsecutivePrimitiveParams)}); err != nil {
			return err
		}
	}

	fmt.Printf("Successfully wrote raw stats to %s\n", filePath)
	return nil
}

func writeAggregatedOutput(stats []stats_collector.FunctionStats) error {
	if outputDir == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputDir = fmt.Sprintf("arit_stats_%s", timestamp)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	linesOfCodeStats := make(map[int]int)
	parameterCountStats := make(map[int]int)
	nestingDepthStats := make(map[int]int)
	messageChainStats := make(map[int]int)
	consecutivePrimitivesStats := make(map[int]int)

	for _, s := range stats {
		linesOfCodeStats[s.LinesOfCode]++
		parameterCountStats[s.ParameterCount]++
		nestingDepthStats[s.MaxNestingDepth]++
		messageChainStats[s.MaxMessageChain]++
		consecutivePrimitivesStats[s.MaxConsecutivePrimitiveParams]++
	}

	if err := writeCSV("lines_of_code", "function_count", linesOfCodeStats); err != nil {
		return err
	}
	if err := writeCSV("parameter_count", "function_count", parameterCountStats); err != nil {
		return err
	}
	if err := writeCSV("nesting_depth", "function_count", nestingDepthStats); err != nil {
		return err
	}
	if err := writeCSV("message_chain_length", "function_count", messageChainStats); err != nil {
		return err
	}
	if err := writeCSV("consecutive_primitives", "function_count", consecutivePrimitivesStats); err != nil {
		return err
	}

	fmt.Printf("Successfully generated stats in directory: %s\n", outputDir)
	return nil
}

func writeCSV(metricName, countName string, data map[int]int) error {
	filePath := filepath.Join(outputDir, fmt.Sprintf("%s_stats.csv", metricName))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create %s stats file: %w", metricName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{metricName, countName}); err != nil {
		return fmt.Errorf("failed to write %s header: %w", metricName, err)
	}

	var keys []int
	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		if err := writer.Write([]string{strconv.Itoa(k), strconv.Itoa(data[k])}); err != nil {
			return fmt.Errorf("failed to write %s data row: %w", metricName, err)
		}
	}
	return nil
}
