// Package main é o ponto de entrada da aplicação ARIT (Analisador de Regras de Integridade de Texto)
// Este analisador estático de código Clojure detecta code smells e problemas de qualidade
package main

import (
	"os"

	"github.com/thlaurentino/arit/cmd"
)

// main é a função principal que inicializa e executa a aplicação ARIT
// Em caso de erro durante a execução, o programa termina com código de saída 1
func main() {
	if err := cmd.Execute(); err != nil {
		// Termina o programa com código de erro se houver falha na execução
		os.Exit(1)
	}
}
