// Package cmd implementa a interface de linha de comando para o ARIT
// usando a biblioteca Cobra para gerenciar comandos e flags
package cmd

import (
	"fmt"
	io "io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reporter"
	"github.com/thlaurentino/arit/internal/rules"
)

var (
	// formatFlag armazena o formato de saída especificado pelo usuário
	formatFlag string
)

// rootCmd define o comando principal da aplicação ARIT
// Aceita arquivos ou diretórios como argumentos e executa a análise estática
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
		// Lista de arquivos Clojure que serão analisados
		filesToAnalyze := []string{}

		// Processa cada argumento fornecido (arquivo ou diretório)
		for _, arg := range args {
			fileInfo, err := os.Stat(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accessing argument %q: %v\n", arg, err)
				continue
			}

			// Se for diretório, busca recursivamente por arquivos Clojure
			if fileInfo.IsDir() {
				cljFiles, err := findClojureFiles(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding Clojure files in directory %q: %v\n", arg, err)
					continue
				}
				filesToAnalyze = append(filesToAnalyze, cljFiles...)
			} else {
				// Verifica se o arquivo tem extensão Clojure válida
				ext := strings.ToLower(filepath.Ext(arg))
				if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
					filesToAnalyze = append(filesToAnalyze, arg)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Skipping non-Clojure file %q\n", arg)
				}
			}
		}

		// Verifica se encontrou arquivos para analisar
		if len(filesToAnalyze) == 0 {
			fmt.Fprintln(os.Stderr, "No Clojure files found to analyze.")
			return nil
		}

		// Determina o diretório de configuração procurando por .git ou go.mod
		configDir := "."
		if len(filesToAnalyze) > 0 {
			firstFileAbs, err := filepath.Abs(filesToAnalyze[0])
			if err == nil {
				parentDir := filepath.Dir(firstFileAbs)
				// Sobe na hierarquia de diretórios procurando pela raiz do projeto
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

		// Carrega a configuração do arquivo .arit.yaml
		cfg, err := config.LoadConfig(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error loading .arit.yaml config from %s: %v. Using default settings.\n", configDir, err)
			// Usa configuração padrão se não conseguir carregar
			cfg = &config.Config{
				EnabledRules: make(map[string]bool),
				RuleConfig:   make(map[string]config.RuleSettings),
			}
		}

		// Configura o formato de saída do relatório
		outputFormat := reporter.ReportFormat(formatFlag)
		allFindings := []*rules.Finding{}

		// Configuração para processamento paralelo dos arquivos
		var wg sync.WaitGroup
		var mu sync.Mutex

		// Analisa cada arquivo em paralelo para melhor performance
		for _, fileToAnalyze := range filesToAnalyze {
			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()
				// Recupera de panics para evitar que um arquivo problemático derrube toda a análise
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC RECOVERED] in goroutine for file '%s': %v", filePath, r)
					}
				}()

				log.Printf("Analyzing file: %s", filePath)
				analysisResult, analyzeErr := analyzer.AnalyzeFile(filePath, cfg)
				if analyzeErr != nil {
					log.Printf("[ERROR Main Goroutine] Error analyzing file '%s': %v", filePath, analyzeErr)
					return
				}

				// Protege o acesso concorrente à lista de findings
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
		// Aguarda todas as goroutines terminarem
		wg.Wait()

		// Gera e exibe o relatório final
		fmt.Fprintf(os.Stderr, "\n--- Analysis Findings (%d) ---\n", len(allFindings))
		rep, err := reporter.NewReporter(outputFormat)
		if err != nil {
			return fmt.Errorf("error creating reporter: %w", err)
		}

		var outputWriter io.Writer = os.Stdout
		err = rep.Report(allFindings, outputWriter)
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		fmt.Fprintln(os.Stderr, "\nAnalysis complete.")
		return nil
	},
}

// Execute executa o comando raiz da aplicação
func Execute() error {
	return rootCmd.Execute()
}

// init configura as flags e opções do comando
func init() {
	// Flag para especificar o formato de saída do relatório
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", string(reporter.FormatText), "Output format (text, json, html, markdown, sarif)")
}

// findClojureFiles busca recursivamente por arquivos Clojure em um diretório
// Retorna uma lista de caminhos para arquivos com extensões .clj, .cljs ou .cljc
func findClojureFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error accessing path %q: %v\n", path, err)
			return nil // Continua a busca mesmo com erros em arquivos específicos
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			// Verifica se é um arquivo Clojure válido
			if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %w", dir, err)
	}

	return files, nil
}

// Declaração vazia para garantir que o tipo rules.Rule está sendo usado
// Isso evita warnings de import não utilizado durante o desenvolvimento
var _ = rules.Rule{}
