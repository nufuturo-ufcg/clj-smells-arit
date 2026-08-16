# Nome dos binários de saída
BINARY_NAME=arit
LSP_BINARY_NAME=arit-lsp

# Diretório de saída
BIN_DIR=bin

# Flags do Go
# -s e -w removem a tabela de símbolos de debug, deixando o binário final (como o do LSP) bem menor.
LDFLAGS=-ldflags="-s -w"

.PHONY: all build build-lsp test clean help

# Target padrão executado ao digitar apenas `make`
all: clean test build build-lsp

## build: Compila o binário padrão da CLI do ARIT
build:
	@echo "==> Compilando CLI padrão do ARIT..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "==> Binário CLI gerado em $(BIN_DIR)/$(BINARY_NAME)"

## build-lsp: Compila a versão ultra-leve e daemonizada exclusiva para o Clojure-LSP
build-lsp:
	@echo "==> Compilando versão otimizada para LSP..."
	@mkdir -p $(BIN_DIR)
	go build -tags lsp $(LDFLAGS) -o $(BIN_DIR)/$(LSP_BINARY_NAME) .
	@echo "==> Binário LSP gerado em $(BIN_DIR)/$(LSP_BINARY_NAME)"

## test: Roda todos os testes unitários do projeto
test:
	@echo "==> Rodando testes unitários..."
	go test -v ./...

## clean: Remove os binários gerados
clean:
	@echo "==> Limpando artefatos de build..."
	rm -rf $(BIN_DIR)
	@echo "==> Limpo."

## help: Exibe esta mensagem de ajuda
help:
	@echo "Comandos disponíveis no Makefile:"
	@sed -n 's/^##//p' $< | column -t -s ':'
