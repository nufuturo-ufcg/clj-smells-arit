// Package rules define as estruturas e interfaces para regras de análise de código
// Contém tipos básicos para findings, severidade e localização de problemas
package rules

import "github.com/thlaurentino/arit/internal/reader"

// Severity define os níveis de severidade para os findings
type Severity string

const (
	// SeverityError   Severity = "ERROR"   // Erros críticos (não usado atualmente)
	SeverityWarning Severity = "WARNING" // Avisos importantes que devem ser corrigidos
	SeverityInfo    Severity = "INFO"    // Informações úteis para melhoria do código
	SeverityHint    Severity = "HINT"    // Sugestões menores de melhoria
)

// Finding representa um problema encontrado durante a análise
// Contém todas as informações necessárias para reportar e localizar o problema
type Finding struct {
	RuleID   string           `json:"rule_id"`  // Identificador único da regra
	Message  string           `json:"message"`  // Descrição do problema encontrado
	Filepath string           `json:"filepath"` // Caminho do arquivo onde foi encontrado
	Location *reader.Location `json:"location"` // Localização exata no código fonte
	Severity Severity         `json:"severity"` // Nível de severidade do problema
}
