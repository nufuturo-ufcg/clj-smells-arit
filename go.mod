// Módulo ARIT - Analisador de Regras de Integridade de Texto para código Clojure
module github.com/thlaurentino/arit

go 1.21.6

require (
	github.com/cespare/goclj v1.2.3 // Parser de código Clojure para análise sintática
	gopkg.in/yaml.v3 v3.0.1         // Biblioteca YAML para arquivos de configuração
)

require github.com/spf13/cobra v1.9.1 // Framework CLI para interface de linha de comando

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect - Dependência do Cobra para Windows
	github.com/spf13/pflag v1.0.6               // indirect - Sistema de flags do Cobra
)
