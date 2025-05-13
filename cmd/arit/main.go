package main

import (
	"flag"
	"fmt"
	io "io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/reporter"
	"github.com/thlaurentino/arit/internal/rules"
)

func findClojureFiles(dir string) ([]string, error) {
	var files []string
	log.Printf("[DEBUG] Searching for Clojure files in: %s\n", dir)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {

			fmt.Fprintf(os.Stderr, "Warning: Error accessing path %q: %v\n", path, err)
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
				log.Printf("[DEBUG] Found Clojure file: %s\n", path)
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %w", dir, err)
	}
	log.Printf("[DEBUG] Found %d Clojure files in %s.\n", len(files), dir)
	return files, nil
}

var _ = rules.Rule{}

func main() {
	fmt.Fprintln(os.Stderr, `
###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############  
                         `)

	formatFlag := flag.String("format", string(reporter.FormatText), "Output format (text, json, html, markdown)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-format=...] <file-or-dir> [file-or-dir...]\n", os.Args[0])
		flag.Usage()
		os.Exit(1)
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
		os.Exit(0)
	}

	configDir := "."
	if len(filesToAnalyze) > 0 {
		firstFileAbs, err := filepath.Abs(filesToAnalyze[0])
		if err == nil {

			parentDir := filepath.Dir(firstFileAbs)
			for parentDir != "/" && parentDir != "." {
				gitPath := filepath.Join(parentDir, ".git")
				modPath := filepath.Join(parentDir, "go.mod")
				gitInfo, gitErr := os.Stat(gitPath)
				modInfo, modErr := os.Stat(modPath)
				if (gitErr == nil && gitInfo.IsDir()) || (modErr == nil && !modInfo.IsDir()) {
					configDir = parentDir
					break
				}
				parentDir = filepath.Dir(parentDir)
			}
			if configDir == "." {
				configDir = filepath.Dir(firstFileAbs)
			}
		}
	}

	cfg, err := config.LoadConfig(configDir)
	if err != nil {

		fmt.Fprintf(os.Stderr, "Warning: Error loading .arit.yaml config from %s: %v. Using default settings.\n", configDir, err)
		cfg = &config.Config{
			EnabledRules: make(map[string]bool),
			RuleConfig:   make(map[string]config.RuleSettings),
		}
	}

	outputFormat := reporter.ReportFormat(*formatFlag)

	allFindings := []*rules.Finding{}

	var wg sync.WaitGroup

	var mu sync.Mutex

	for _, fileToAnalyze := range filesToAnalyze {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					log.Printf("[PANIC RECOVERED] in goroutine for file '%s': %v", filePath, r)

				}
			}()

			//	fmt.Fprintf(os.Stderr, "[DEBUG Main Goroutine - ENTRY VIA FPRINTF] Goroutine started. FilePath raw: %s\n", filePath)

			isTargetFileForGoroutineLog := strings.HasSuffix(filePath, "smells/long_function.clj")

			//	fmt.Fprintf(os.Stderr, "[DEBUG Main Goroutine - PRE-PARSE VIA FPRINTF] About to call ParseFile for: %s\n", filePath)

			rawTree, err := reader.ParseFile(filePath)
			if err != nil {
				log.Printf("[ERROR Main Goroutine] Error parsing file '%s': %v", filePath, err)
				return
			}
			if rawTree == nil && isTargetFileForGoroutineLog {
				//	log.Printf("[DEBUG Main Goroutine - CHECK] ParseFile returned a nil rawTree for: '%s'", filePath)
			}

			if isTargetFileForGoroutineLog {
				log.Printf("[DEBUG Main Goroutine - STEP] ParseFile (raw) completed for: '%s'. rawTree is nil: %t", filePath, rawTree == nil)
			}

			richRootNodes, commentNodes := reader.BuildRichTree(rawTree)

			if isTargetFileForGoroutineLog {
				log.Printf("[DEBUG Main Goroutine - STEP] BuildRichTree completed for: '%s'. Got %d rich root nodes (nil: %t) and %d comment nodes (nil: %t).",
					filePath, len(richRootNodes), richRootNodes == nil, len(commentNodes), commentNodes == nil)
			}

			log.Printf("Analyzing file: %s", filePath)

			if isTargetFileForGoroutineLog {
				log.Printf("[DEBUG Main Goroutine - ACTION] Calling analyzer.AnalyzeFile for: '%s'", filePath)
			}

			analysisResult, analyzeErr := analyzer.AnalyzeFile(filePath, cfg)
			if analyzeErr != nil {
				log.Printf("[ERROR Main Goroutine] Error analyzing file '%s': %v", filePath, analyzeErr)
				return
			}

			mu.Lock()
			if analysisResult.Findings != nil {
				for _, finding := range analysisResult.Findings {
					findingCopy := finding
					allFindings = append(allFindings, &findingCopy)
				}
			}
			mu.Unlock()

		}(fileToAnalyze)
	}

	wg.Wait()

	fmt.Fprintf(os.Stderr, "\n--- Analysis Findings (%d) ---\n", len(allFindings))

	rep, err := reporter.NewReporter(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating reporter: %v\n", err)
		os.Exit(1)
	}

	var outputWriter io.Writer = os.Stdout
	err = rep.Report(allFindings, outputWriter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "\nAnalysis complete.")
}
