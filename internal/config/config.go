// Package config gerencia a configuração da aplicação ARIT
// Carrega configurações do arquivo .arit.yaml e fornece métodos para acessar as configurações das regras
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// configFileName define o nome padrão do arquivo de configuração
const configFileName = ".arit.yaml"

// Config representa a estrutura principal de configuração do ARIT
// Contém as regras habilitadas e suas configurações específicas
type Config struct {
	EnabledRules map[string]bool         `yaml:"enabled-rules"` // Mapa de regras e se estão habilitadas
	RuleConfig   map[string]RuleSettings `yaml:"rule-config"`   // Configurações específicas por regra
}

// RuleSettings representa as configurações específicas de uma regra
// Permite flexibilidade para diferentes tipos de configuração por regra
type RuleSettings map[string]interface{}

// LoadConfig carrega a configuração a partir do arquivo .arit.yaml
// Busca o arquivo a partir do diretório especificado, subindo na hierarquia se necessário
func LoadConfig(startDir string) (*Config, error) {
	filePath, found := findConfigFile(startDir)
	if !found {
		// Retorna configuração padrão se não encontrar arquivo de configuração
		return &Config{
			EnabledRules: make(map[string]bool),
			RuleConfig:   make(map[string]RuleSettings),
		}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", filePath, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file %s: %w", filePath, err)
	}

	// Inicializa mapas se estiverem nil para evitar panics
	if config.EnabledRules == nil {
		config.EnabledRules = make(map[string]bool)
	}
	if config.RuleConfig == nil {
		config.RuleConfig = make(map[string]RuleSettings)
	}

	return &config, nil
}

// findConfigFile busca o arquivo de configuração .arit.yaml
// Começa no diretório especificado e sobe na hierarquia até encontrar o arquivo ou chegar na raiz
func findConfigFile(startDir string) (string, bool) {
	dir := startDir
	for {
		filePath := filepath.Join(dir, configFileName)
		if _, err := os.Stat(filePath); err == nil {
			// Arquivo encontrado
			return filePath, true
		} else if !os.IsNotExist(err) {
			// Erro diferente de "arquivo não existe" - pode ser permissão, etc.
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Chegou na raiz do sistema de arquivos
			break
		}
		dir = parentDir
	}
	return "", false
}

// GetRuleSettingBool obtém uma configuração booleana específica de uma regra
// Retorna o valor padrão se a configuração não existir ou não for um booleano
func (c *Config) GetRuleSettingBool(ruleID string, key string, defaultValue bool) bool {
	if ruleSettings, ok := c.RuleConfig[ruleID]; ok {
		if value, ok := ruleSettings[key]; ok {
			if boolValue, ok := value.(bool); ok {
				return boolValue
			}
			// Valor existe mas não é booleano - usa valor padrão
		}
	}
	return defaultValue
}

// GetRuleSettingInt obtém uma configuração inteira específica de uma regra
// Suporta conversão de float64 para int (comum em YAML)
// Retorna o valor padrão se a configuração não existir ou não for numérica
func (c *Config) GetRuleSettingInt(ruleID string, key string, defaultValue int) int {
	if ruleSettings, ok := c.RuleConfig[ruleID]; ok {
		if value, ok := ruleSettings[key]; ok {
			// Tenta diferentes tipos numéricos que podem vir do YAML
			switch v := value.(type) {
			case int:
				return v
			case float64:
				return int(v) // YAML frequentemente parseia números como float64
			}
			// Valor existe mas não é numérico - usa valor padrão
		}
	}
	return defaultValue
}

// IsRuleEnabled verifica se uma regra específica está habilitada
// Retorna o valor padrão se a regra não estiver explicitamente configurada
func (c *Config) IsRuleEnabled(ruleID string, defaultEnabled bool) bool {
	enabled, ok := c.EnabledRules[ruleID]
	if !ok {
		return defaultEnabled // Usa valor padrão se não configurado
	}
	return enabled
}
