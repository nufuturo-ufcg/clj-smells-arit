package rules

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type MultipleEvaluationInMacrosRule struct {
	Rule
}

func (r *MultipleEvaluationInMacrosRule) Meta() Rule {
	return r.Rule
}

func (r *MultipleEvaluationInMacrosRule) extractParameters(paramsNode *reader.RichNode) map[string]int {
   parameters := make(map[string]int)

   if paramsNode == nil || paramsNode.Type != reader.NodeVector {
       return parameters
   }

   for _, currentParam := range paramsNode.Children {
       if currentParam.Type == reader.NodeSymbol {
           if currentParam.Value == "_" ||
               strings.HasPrefix(currentParam.Value, ".") ||
               strings.Contains(currentParam.Value, "/") || currentParam.Value == "&"{
               continue
           }
           parameters[currentParam.Value] = 0
   		}
	}
	return parameters
}

func (r *MultipleEvaluationInMacrosRule) countParametersUses(node *reader.RichNode, count map[string]int) map[string]int {

	for _, child := range node.Children {

    	if child.Type == reader.NodeSymbol {
			currentSymbol := child.Value

			if strings.HasSuffix(currentSymbol, "#"){
				continue
			} 

			if strings.HasPrefix(child.Value, "~") {
				currentSymbol = strings.ReplaceAll(child.Value, "~", "")
			}

			if _, ok := count[currentSymbol]; ok {
					count[currentSymbol]++
			}
        } 
		r.countParametersUses(child, count)
	}
	return count
}

func (r *MultipleEvaluationInMacrosRule) multipleEvaluation(node *reader.RichNode) string {

	parametersCalls := r.countParametersUses(node.Children[3], r.extractParameters(node.Children[2]))

    for mapKey, keyValue := range parametersCalls {
        if keyValue > 1 {
            continue
        } else {
			delete(parametersCalls, mapKey)
		}
    }

	parameters := slices.Collect(maps.Keys(parametersCalls))
	return strings.Join(parameters, ", ")
}

func (r *MultipleEvaluationInMacrosRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {
   if node.Type == reader.NodeList && len(node.Children) > 0 &&
       node.Children[0].Type == reader.NodeSymbol &&
       node.Children[0].Value == "defmacro" {
	   smell := r.multipleEvaluation(node)
       if len(smell) != 0{
           return &Finding{
               RuleID:   r.ID,
               Message:  fmt.Sprintf("The macro %s presents multiple calls to the input arguments %v without defining temporary local variables.", node.Children[1].Value, smell),
               Filepath: filepath,
               Location: node.Location,
               Severity: r.Severity,
           }
       }
   }
   return nil
}

func init() {
	defaultRule := &MultipleEvaluationInMacrosRule{
		Rule: Rule{
			ID:          "multiple-evaluation-in-macros",
			Name:        "Multiple Evaluation in Macros",
			Description: "Inserting macro input arguments more than once without first binding them to a local, temporary variable violates macro best practices and leads to hidden side effects.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
