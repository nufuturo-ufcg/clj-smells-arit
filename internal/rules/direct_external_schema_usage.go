package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

const (
	DirectExternalSchemaUsageRuleID = "external-data-coupling:direct-external-schema-usage"

	DirectExternalSchemaUsageRuleName = "Direct External Schema Usage"
)

type DirectExternalSchemaUsageRule struct {
	Rule
}

func newDirectExternalSchemaUsageRule() *DirectExternalSchemaUsageRule {
	desc := "Detects functions that directly consume data structures resembling an external schema " +
		"(e.g., using snake_case keys like :user_id or :item_name) " +
		"without an apparent transformation step to an internal application-specific model. " +
		"This can lead to tight coupling with external data formats."

	return &DirectExternalSchemaUsageRule{
		Rule: Rule{
			ID:          DirectExternalSchemaUsageRuleID,
			Name:        DirectExternalSchemaUsageRuleName,
			Description: desc,
			Severity:    SeverityWarning,
		},
	}
}

func (r *DirectExternalSchemaUsageRule) Meta() Rule {
	return r.Rule
}

func (r *DirectExternalSchemaUsageRule) Check(node *reader.RichNode, fileContext map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList {
		return nil
	}

	children := node.Children
	if len(children) == 0 {
		return nil
	}

	if children[0].Type == reader.NodeSymbol && children[0].Value == "defn" {
		if len(children) < 4 {
			return nil
		}

		funcNameNode := children[1]
		paramsNode := children[2]

		if funcNameNode.Type != reader.NodeSymbol || paramsNode.Type != reader.NodeVector {
			return nil
		}
		funcName := funcNameNode.Value

		if r.isTransformerFunction(funcName, fileContext) {
			return nil
		}

		var paramSymbols []string
		for _, pNode := range paramsNode.Children {
			if pNode.Type == reader.NodeSymbol {
				val := pNode.Value
				if !strings.Contains(val, "&") && !strings.Contains(val, ".") && !strings.Contains(val, "#") && val != "_" {
					paramSymbols = append(paramSymbols, val)
				}
			}
		}

		if len(paramSymbols) == 0 {
			return nil
		}

		for i := 3; i < len(children); i++ {
			bodyForm := children[i]
			var defnLine int
			if node.Location != nil {
				defnLine = node.Location.StartLine
			}
			finding := r.findExternalKeyAccessInForm(bodyForm, funcName, paramSymbols, filepath, fileContext, defnLine)
			if finding != nil {
				return finding
			}
		}
	}
	return nil
}

func (r *DirectExternalSchemaUsageRule) findExternalKeyAccessInForm(
	form *reader.RichNode,
	currentFuncName string,
	funcParamSymbols []string,
	filepath string,
	fileContext map[string]interface{},
	defnLine int) *Finding {

	if form.Type == reader.NodeList && len(form.Children) > 0 {
		callChildren := form.Children
		firstElement := callChildren[0]

		if len(callChildren) == 2 && firstElement.Type == reader.NodeKeyword && callChildren[1].Type == reader.NodeSymbol {
			accessedKey := firstElement.Value
			accessedParam := callChildren[1].Value

			for _, pSym := range funcParamSymbols {
				if pSym == accessedParam && r.isExternalKey(accessedKey) {
					message := fmt.Sprintf("Function '%s' (defined at line %d) directly accesses potentially external key '%s' on parameter '%s'. Consider transforming to an internal model.", currentFuncName, defnLine, accessedKey, accessedParam)
					return &Finding{
						RuleID:   r.ID,
						Message:  message,
						Severity: r.Severity,
						Filepath: filepath,
						Location: firstElement.Location,
					}
				}
			}
		}

		if firstElement.Type == reader.NodeSymbol {
			resolvedGet := dsrcResolveSymbol(firstElement.Value, fileContext)
			isGet := resolvedGet == "clojure.core/get"
			isGetIn := resolvedGet == "clojure.core/get-in"

			if (isGet && len(callChildren) >= 3) || (isGetIn && len(callChildren) >= 3) {
				paramNode := callChildren[1]
				keyNodeOrPathNode := callChildren[2]

				if paramNode.Type == reader.NodeSymbol {
					accessedParam := paramNode.Value
					isParamTracked := false
					for _, pSym := range funcParamSymbols {
						if pSym == accessedParam {
							isParamTracked = true
							break
						}
					}

					if isParamTracked {
						var keysToCheck []*reader.RichNode
						if isGet && keyNodeOrPathNode.Type == reader.NodeKeyword {
							keysToCheck = append(keysToCheck, keyNodeOrPathNode)
						} else if isGetIn && keyNodeOrPathNode.Type == reader.NodeVector {
							for _, pathKeyNode := range keyNodeOrPathNode.Children {
								if pathKeyNode.Type == reader.NodeKeyword {
									keysToCheck = append(keysToCheck, pathKeyNode)
								}
							}
						}

						for _, keyNode := range keysToCheck {
							accessedKey := keyNode.Value
							if r.isExternalKey(accessedKey) {
								verb := "get"
								if isGetIn {
									verb = "get-in"
								}
								message := fmt.Sprintf("Function '%s' (defined at line %d) uses '%s' with potentially external key '%s' on parameter '%s'. Consider transforming to an internal model.", currentFuncName, defnLine, verb, accessedKey, accessedParam)
								return &Finding{
									RuleID:   r.ID,
									Message:  message,
									Severity: r.Severity,
									Filepath: filepath,
									Location: keyNode.Location,
								}
							}
						}
					}
				}
			}
		}

		for _, childForm := range callChildren {
			if finding := r.findExternalKeyAccessInForm(childForm, currentFuncName, funcParamSymbols, filepath, fileContext, defnLine); finding != nil {
				return finding
			}
		}
	} else if form.Type == reader.NodeVector || form.Type == reader.NodeMap || form.Type == reader.NodeSet {
		for _, childForm := range form.Children {
			if finding := r.findExternalKeyAccessInForm(childForm, currentFuncName, funcParamSymbols, filepath, fileContext, defnLine); finding != nil {
				return finding
			}
		}
	}
	return nil
}

func (r *DirectExternalSchemaUsageRule) isExternalKey(keyStr string) bool {
	if !strings.HasPrefix(keyStr, ":") {
		return false
	}
	keyName := keyStr[1:]

	hasUnderscore := strings.Contains(keyName, "_")
	if !hasUnderscore {
		return false
	}

	if strings.HasPrefix(keyName, "_") || strings.HasSuffix(keyName, "_") {
		return false
	}

	isScreamingSnake := true
	for _, char := range keyName {
		if char >= 'a' && char <= 'z' {
			isScreamingSnake = false
			break
		}
	}
	if isScreamingSnake && strings.ContainsAny(keyName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return false
	}

	parts := strings.Split(keyName, "_")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	return true
}

func (r *DirectExternalSchemaUsageRule) isTransformerFunction(funcName string, fileContext map[string]interface{}) bool {
	normalizedFuncName := strings.ToLower(funcName)
	if strings.Contains(normalizedFuncName, "transform") ||
		strings.Contains(normalizedFuncName, "parse") ||
		strings.Contains(normalizedFuncName, "to-internal") ||
		strings.Contains(normalizedFuncName, "map-external") ||
		strings.Contains(normalizedFuncName, "external-to") {
		return true
	}
	return false
}

func dsrcResolveSymbol(symbolName string, _ map[string]interface{}) string {

	if strings.Contains(symbolName, "/") || strings.Contains(symbolName, ".") || strings.HasPrefix(symbolName, "&") || strings.HasPrefix(symbolName, "#") {
		return symbolName
	}

	if symbolName == "get" || symbolName == "get-in" {
		return "clojure.core/" + symbolName
	}

	return symbolName
}

func dsrcIsCoreSymbol(symbolName string) bool {
	coreSymbols := map[string]struct{}{
		"get": {}, "assoc": {}, "dissoc": {}, "merge": {}, "select-keys": {},
		"map": {}, "filter": {}, "reduce": {}, "fn": {}, "if": {}, "let": {}, "loop": {},
		"get-in": {},
	}
	_, isCore := coreSymbols[symbolName]
	return isCore
}

func init() {
	RegisterRule(newDirectExternalSchemaUsageRule())
}
