# Nome dos binários de saída
BINARY_NAME=arit
ARITD_BINARY_NAME=aritd

# Diretório de saída
BIN_DIR=bin

# Flags do Go para o Daemon
LDFLAGS=-ldflags="-s -w"

.PHONY: all build build-aritd test clean help

# Target padrão executado ao digitar apenas `make`
all: clean test build build-aritd

## build: Compila o binário padrão da CLI do ARIT
build:
	@echo "==> Compilando CLI padrão do ARIT..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "==> Binário CLI gerado em $(BIN_DIR)/$(BINARY_NAME)"

## build-aritd: Compila a versão daemon (aritd)
build-aritd:
	@echo "==> Compilando versão daemon (aritd)..."
	@mkdir -p $(BIN_DIR)
	go build -tags aritd $(LDFLAGS) -o $(BIN_DIR)/$(ARITD_BINARY_NAME) .
	@echo "==> Binário daemon gerado em $(BIN_DIR)/$(ARITD_BINARY_NAME)"

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
