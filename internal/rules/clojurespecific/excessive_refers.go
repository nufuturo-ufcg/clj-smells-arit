package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ExcessiveRefersRule struct {
	rules.Rule
	MaxExplicitRefers int `json:"max_explicit_refers" yaml:"max_explicit_refers"`
}

func (r *ExcessiveRefersRule) Meta() rules.Rule {
	return r.Rule
}

// referredByPlatform collects the distinct names referred by :refer and :only,
// keeping common references ("*") separate from reader-conditional platforms.
// A name is counted once per platform even when it is repeated, and a
// conditional element inside a refer vector belongs only to its platform.
func referredByPlatform(nodes []*reader.RichNode, platform string, sets map[string]map[string]bool) {
	add := func(platform, name string) {
		if sets[platform] == nil {
			sets[platform] = map[string]bool{}
		}
		sets[platform][name] = true
	}
	var addVector func(vector *reader.RichNode, platform string)
	addVector = func(vector *reader.RichNode, platform string) {
		for _, element := range vector.Children {
			switch {
			case element == nil:
			case element.Type == reader.NodeSymbol:
				add(platform, element.Value)
			case element.Type == reader.NodeReaderCond || element.Type == reader.NodeReaderCondSplice:
				for branch := 1; branch < len(element.Children); branch += 2 {
					branchPlatform := conditionalPlatform(element.Children[branch-1], platform)
					value := element.Children[branch]
					if value != nil && value.Type == reader.NodeSymbol {
						add(branchPlatform, value.Value)
					} else if value != nil && value.Type == reader.NodeVector {
						addVector(value, branchPlatform)
					}
				}
			}
		}
	}
	for i, child := range nodes {
		if child == nil || child.Type == reader.NodeReaderDiscard {
			continue
		}
		if child.Type == reader.NodeReaderCond || child.Type == reader.NodeReaderCondSplice {
			for branch := 1; branch < len(child.Children); branch += 2 {
				referredByPlatform([]*reader.RichNode{child.Children[branch]}, conditionalPlatform(child.Children[branch-1], platform), sets)
			}
			continue
		}
		if child.Type == reader.NodeKeyword && (child.Value == ":refer" || child.Value == ":only") &&
			i+1 < len(nodes) && nodes[i+1] != nil && nodes[i+1].Type == reader.NodeVector {
			addVector(nodes[i+1], platform)
			continue
		}
		if child.Type == reader.NodeVector && i > 0 && nodes[i-1] != nil && nodes[i-1].Type == reader.NodeKeyword &&
			(nodes[i-1].Value == ":refer" || nodes[i-1].Value == ":only") {
			continue
		}
		if len(child.Children) > 0 {
			referredByPlatform(child.Children, platform, sets)
		}
	}
}

// conditionalPlatform names a reader-conditional branch; a branch nested in
// another platform's branch stays in that platform.
func conditionalPlatform(key *reader.RichNode, outer string) string {
	if outer != "*" {
		return outer
	}
	if key != nil && key.Type == reader.NodeKeyword {
		return key.Value
	}
	return "*"
}

// maxPlatformReferenceCount is the largest number of distinct names any one
// platform refers: the common names plus that platform's own.
func maxPlatformReferenceCount(sets map[string]map[string]bool) int {
	maximum := len(sets["*"])
	for platform, names := range sets {
		if platform == "*" {
			continue
		}
		union := len(sets["*"])
		for name := range names {
			if !sets["*"][name] {
				union++
			}
		}
		if union > maximum {
			maximum = union
		}
	}
	return maximum
}

func namespaceDeclaredName(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	for _, child := range node.Children[1:] {
		if child != nil && child.Type == reader.NodeSymbol {
			return child.Value
		}
	}
	return ""
}

func (r *ExcessiveRefersRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	if node.Children[0].Value == "ns" {
		sets := map[string]map[string]bool{}
		referredByPlatform(node.Children[1:], "*", sets)
		totalExplicitRefers := maxPlatformReferenceCount(sets)

		if totalExplicitRefers >= r.MaxExplicitRefers {
			return &rules.Finding{
				RuleID: r.ID,
				Message: fmt.Sprintf(
					"Namespace `%s` explicitly refers %d Vars, meeting the configured threshold of %d. The default threshold (24) was calibrated as the mean plus two standard deviations across 800 important repositories. This is a proven excessive-refers outlier under the calibrated rule.",
					namespaceDeclaredName(node), totalExplicitRefers, r.MaxExplicitRefers,
				),
				Filepath:   filepath,
				Location:   node.Location,
				Severity:   r.Severity,
				Confidence: rules.ConfidenceProven,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &ExcessiveRefersRule{
		Rule: rules.Rule{
			ID:          "excessive-refers",
			Name:        "Excessive Refers",
			Description: "Detects proven statistical outliers in the total number of Vars explicitly imported through :refer [...] or :use ... :only [...]. The default inclusive threshold of 24 was calibrated as the mean plus two standard deviations across 800 important repositories. Unrestricted imports such as :refer :all belong to implicit-namespace-dependencies.",
			Severity:    rules.SeverityWarning,
		},
		MaxExplicitRefers: 24,
	}
	rules.RegisterRule(defaultRule)
}
