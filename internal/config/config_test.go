package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigReadsSemanticContracts(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, configFileName)
	contents := []byte("semantic-contracts:\n  sample/read:\n    effects: [io]\n    returns-type: string\n")
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	contract, ok := loaded.SemanticContracts["sample/read"]
	if !ok || contract["returns-type"] != "string" {
		t.Fatalf("expected semantic contract to be loaded, got %#v", loaded.SemanticContracts)
	}
}
