package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// markerProtocolRule detecta (defprotocol XYZ) sem nenhum método declarado.
// Um defprotocol vazio é um anti-padrão herdado de Java (Marker Interface),
// que introduz sobrecarga do sistema de protocolos da JVM sem valor funcional.
// A alternativa idiomática em Clojure é usar metadados, chaves de mapa ou Clojure Spec.
type markerProtocolRule struct {
	rules.Rule
}

func (r *markerProtocolRule) Meta() rules.Rule {
	return r.Rule
}

func (r *markerProtocolRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	// Nó deve ser uma lista
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	first := node.Children[0]
	if first.Type != reader.NodeSymbol || first.Value != "defprotocol" {
		return nil
	}

	// defprotocol precisa ter pelo menos o nome do protocolo
	if len(node.Children) < 2 {
		return nil
	}

	protocolName := ""
	if node.Children[1].Type == reader.NodeSymbol {
		protocolName = node.Children[1].Value
	}

	// Conta quantos filhos são declarações de método.
	// Um método no defprotocol é uma lista: (method-name [args] ...)
	// Pode haver docstring (NodeString) logo após o nome — não conta como método
	methodCount := 0
	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		// Docstring — não é método
		if child.Type == reader.NodeString {
			continue
		}
		// Declaração de método: (method-name [args] ...)
		if child.Type == reader.NodeList && len(child.Children) >= 1 {
			if child.Children[0].Type == reader.NodeSymbol {
				methodCount++
			}
		}
	}

	// Se não há nenhum método → marker protocol
	if methodCount == 0 {
		name := protocolName
		if name == "" {
			name = "anonymous"
		}
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Marker protocol: `(defprotocol %s)` has no methods. "+
					"Empty protocols are an OOP anti-pattern in Clojure. "+
					"Use metadata (^:marker), map keys, or Clojure Spec instead.",
				name,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func init() {
	rules.RegisterRule(&markerProtocolRule{
		Rule: rules.Rule{
			ID:          "marker-protocol",
			Name:        "Marker Protocol",
			Description: "Detects defprotocol with no methods, used only as a type tag. Empty protocols are an OOP anti-pattern in Clojure; use metadata, map keys, or Clojure Spec to mark types.",
			Severity:    rules.SeverityInfo,
		},
	})
}
