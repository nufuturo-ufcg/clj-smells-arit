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
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	if !rules.CallResolvesTo(node, "clojure.core/defprotocol") {
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

	methodCount := 0
	for i := 2; i < len(node.Children); i++ {
		if isProtocolMethodDeclaration(node.Children[i]) {
			methodCount++
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

// goclj exposes metadata such as (^:export method [args]) as a leading
// keyword in the method list. A valid declaration therefore is not limited to
// lists whose first child is the method symbol: it is a list containing a
// method symbol followed by at least one arity vector.
func isProtocolMethodDeclaration(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList {
		return false
	}
	methodSeen := false
	for _, child := range node.Children {
		if !methodSeen {
			if child.Type == reader.NodeSymbol {
				methodSeen = true
			}
			continue
		}
		if child.Type == reader.NodeVector {
			return true
		}
	}
	return false
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
