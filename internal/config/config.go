package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = ".arit.yaml"

type Config struct {
	AnalyzeTests  bool                    `yaml:"analyze-tests"`
	EnabledRules  map[string]bool         `yaml:"enabled-rules"`
	EnabledGroups map[string]bool         `yaml:"enabled-groups"`
	RuleConfig    map[string]RuleSettings `yaml:"rule-config"`
}

type RuleSettings map[string]interface{}

func LoadConfig(startDir string) (*Config, error) {
	filePath, found := findConfigFile(startDir)
	if !found {
		// Fallback 1: Try current working directory
		if cwd, err := os.Getwd(); err == nil {
			filePath, found = findConfigFile(cwd)
		}
	}
	if !found {
		// Fallback 2: Try user home directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			homeFilePath := filepath.Join(homeDir, configFileName)
			if _, err := os.Stat(homeFilePath); err == nil {
				filePath = homeFilePath
				found = true
			}
		}
	}

	if !found {
		return &Config{
			EnabledRules:  make(map[string]bool),
			EnabledGroups: make(map[string]bool),
			RuleConfig:    make(map[string]RuleSettings),
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

	if config.EnabledRules == nil {
		config.EnabledRules = make(map[string]bool)
	}
	if config.EnabledGroups == nil {
		config.EnabledGroups = make(map[string]bool)
	}
	if config.RuleConfig == nil {
		config.RuleConfig = make(map[string]RuleSettings)
	}

	return &config, nil
}

func findConfigFile(startDir string) (string, bool) {
	dir := startDir
	for {
		filePath := filepath.Join(dir, configFileName)
		if _, err := os.Stat(filePath); err == nil {

			return filePath, true
		} else if !os.IsNotExist(err) {

		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {

			break
		}
		dir = parentDir
	}
	return "", false
}

func (c *Config) GetRuleSettingBool(ruleID string, key string, defaultValue bool) bool {
	if ruleSettings, ok := c.RuleConfig[ruleID]; ok {
		if value, ok := ruleSettings[key]; ok {
			if boolValue, ok := value.(bool); ok {
				return boolValue
			}

		}
	}
	return defaultValue
}

func (c *Config) GetRuleSettingInt(ruleID string, key string, defaultValue int) int {
	if ruleSettings, ok := c.RuleConfig[ruleID]; ok {
		if value, ok := ruleSettings[key]; ok {

			switch v := value.(type) {
			case int:
				return v
			case float64:
				return int(v)
			}

		}
	}
	return defaultValue
}

func (c *Config) IsRuleEnabled(ruleID string, defaultEnabled bool) bool {
	enabled, ok := c.EnabledRules[ruleID]
	if !ok {
		return defaultEnabled
	}
	return enabled
}
