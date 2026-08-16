# Nome dos binários de saída
BINARY_NAME=arit

# Diretório de saída
BIN_DIR=bin

.PHONY: all build test clean help

# Target padrão executado ao digitar apenas `make`
all: clean test build

## build: Compila o binário padrão da CLI do ARIT
build:
	@echo "==> Compilando CLI padrão do ARIT..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "==> Binário CLI gerado em $(BIN_DIR)/$(BINARY_NAME)"

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
