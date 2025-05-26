package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

// RedundantDoBlockRule identifica blocos `do` explícitos que são desnecessários
// porque sua forma pai já fornece um `do` implícito.
type RedundantDoBlockRule struct{}

// Meta retorna os metadados da regra.
func (r *RedundantDoBlockRule) Meta() Rule {
	return Rule{
		ID:          "redundant-do-block",
		Description: "Verifica blocos `do` redundantes dentro de formas que já implicam execução sequencial de seus corpos ou cláusulas.",
		Severity:    SeverityInfo,
	}
}

// getFnBodyStartIndex calcula o índice onde o corpo real de uma forma similar a função começa.
// Pula o símbolo principal, nome (se houver), docstring (se houver), mapa de atributos (se houver),
// e o(s) vetor(es) de parâmetros/lista de aridades.
// Retorna o índice da primeira expressão no corpo.
func getFnBodyStartIndex(parentChildren []*reader.RichNode, parentSymbol string) int {
	isDefnLike := parentSymbol == "defn" || parentSymbol == "defn-" || parentSymbol == "defmacro" || parentSymbol == "defmethod"
	isFnLike := parentSymbol == "fn"

	currentIndex := 1 // Índice do primeiro elemento após o símbolo principal (ex: "defn", "fn")

	if isDefnLike {
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeSymbol { // Nome da função/macro
			currentIndex++
		}
	} else if isFnLike {
		// Nome opcional da fn para (fn nome [params] corpo...)
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeSymbol &&
			currentIndex+1 < len(parentChildren) &&
			(parentChildren[currentIndex+1].Type == reader.NodeVector || parentChildren[currentIndex+1].Type == reader.NodeList) {
			currentIndex++
		}
	}

	if isDefnLike { // defn, defmacro etc. podem ter docstring e mapa de atributos
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeString { // Docstring
			currentIndex++
		}
		if currentIndex < len(parentChildren) && parentChildren[currentIndex].Type == reader.NodeMap { // Mapa de atributos
			currentIndex++
		}
	}

	// Neste ponto, parentChildren[currentIndex] deve ser o vetor de parâmetros ou a primeira forma de aridade (uma lista).
	// As formas do corpo real começam após isso.
	if currentIndex < len(parentChildren) &&
		(parentChildren[currentIndex].Type == reader.NodeVector || parentChildren[currentIndex].Type == reader.NodeList) {
		return currentIndex + 1 // Corpo começa após params/lista de aridades
	}
	// Isso pode ser alcançado se a forma estiver malformada ou não tiver params/corpo,
	// ou se for uma forma fn simples como (fn corpo...) que não é padrão.
	// Para (fn nome? [params] corpo...) ou (fn nome? ([p1] b1) ([p2] b2)), isso aponta após params/aridades.
	return currentIndex
}

// Check executa a verificação da regra para blocos `do` redundantes.
func (r *RedundantDoBlockRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "do" {
		return nil
	}

	// Agora sabemos que 'node' é uma forma (do ...)
	var parent *reader.RichNode
	if pVal, ok := context["parent"]; ok {
		if pNode, pOk := pVal.(*reader.RichNode); pOk {
			parent = pNode
		}
	}

	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) == 0 {
		return nil
	}

	parentFirstElement := parent.Children[0]
	if parentFirstElement.Type != reader.NodeSymbol {
		// Pode ser parte de uma fn multi-aridade, ex: parent é '([x] (do ...))'
		// ou uma cláusula try-catch como (catch Exception e (do ...))
		// ou (finally (do ...))
		// Trata esses casos específicos onde o pai direto não começa com um símbolo conhecido.
		if parentFirstElement.Type == reader.NodeVector { // Provavelmente uma definição de aridade como ([params] corpo...)
			// Parent é ( [params] expr1 expr2 ...), node é uma das exprN
			// Se esta lista de aridade é ela mesma filha de fn/defn, é um do implícito.
			// Verificamos se 'node' (a forma 'do') é uma das expressões do corpo.
			// O vetor de params está no índice 0 de parent.Children.
			// Então, se 'node' está no índice > 0 em parent.Children, está no corpo desta aridade.
			doNodeIndexInArity := -1
			for i, child := range parent.Children {
				if child == node {
					doNodeIndexInArity = i
					break
				}
			}
			if doNodeIndexInArity > 0 { // 'do' é uma expressão do corpo de uma aridade
				var grandParent *reader.RichNode
				if gpVal, gpOk := context["grandparent"]; gpOk { // Assumindo que grandparent também é passado no contexto se necessário
					if gpNode, gpNodeOk := gpVal.(*reader.RichNode); gpNodeOk {
						grandParent = gpNode
					}
				}

				if grandParent != nil && grandParent.Type == reader.NodeList && len(grandParent.Children) > 0 &&
					grandParent.Children[0].Type == reader.NodeSymbol {
					gpSymbol := grandParent.Children[0].Value
					switch gpSymbol {
					case "fn", "defn", "defn-", "defmacro", "defmethod", "proxy", "reify", "deftype", "defrecord", "extend-protocol", "extend-type":
						// A aridade em si fornece um do implícito para seu corpo.
						return r.createFinding(node, parent, "lista de aridade", filepath)
					}
				}
			}
		}
		return nil // Padrão: se parent não começa com símbolo, não é um padrão reconhecido para esta regra.
	}
	parentSymbol := parentFirstElement.Value

	doNodeIndex := -1
	for i, child := range parent.Children {
		if child == node {
			doNodeIndex = i
			break
		}
	}
	if doNodeIndex == -1 {
		return nil
	}

	isRedundant := false
	redundantInForm := parentSymbol

	switch parentSymbol {
	case "let", "loop", "letfn", "binding", "with-local-vars", "with-open", "with-out-str", "with-in-str", "locking", "future", "promise", "testing", "comment", "doto", "doseq", "dotimes":
		// Redundante se 'do' está na parte do corpo (após símbolo no 0 e vetor de bindings no 1)
		if doNodeIndex >= 2 {
			isRedundant = true
		}
	case "when", "when-not":
		// Redundante se 'do' está na parte do corpo (após símbolo no 0 e expressão de teste no 1)
		if doNodeIndex >= 2 {
			isRedundant = true
		}
	case "when-let", "when-some":
		// Redundante se 'do' está na parte do corpo (após símbolo no 0 e vetor de binding no 1)
		if doNodeIndex >= 2 {
			isRedundant = true
		}
	case "if", "if-not": // if-not é como if com teste invertido
		// Redundante se 'do' é o ramo 'then' (índice 2) ou 'else' (índice 3). Teste está no índice 1.
		if doNodeIndex == 2 || (doNodeIndex == 3 && len(parent.Children) > 3) {
			isRedundant = true
		}
	case "if-let", "if-some":
		// Redundante se 'do' é o ramo 'then' (índice 2) ou 'else' (índice 3). Bindings estão no índice 1.
		if doNodeIndex == 2 || (doNodeIndex == 3 && len(parent.Children) > 3) {
			isRedundant = true
		}
	case "fn", "defn", "defn-", "defmacro", "defmethod", "proxy", "reify", "deftype", "defrecord", "extend-protocol", "extend-type":
		bodyStartIndex := getFnBodyStartIndex(parent.Children, parentSymbol)
		if doNodeIndex >= bodyStartIndex {
			isRedundant = true
		}
	case "try":
		// (try corpo... catch... finally...)
		// 'do' é redundante se é uma forma de nível superior no bloco try principal.
		// O bloco try principal pode conter múltiplas expressões.
		// Cláusulas catch e finally são formas de lista separadas.
		isTryBody := true
		for i := 1; i < doNodeIndex; i++ { // Verifica se formas anteriores são catch/finally
			if parent.Children[i].Type == reader.NodeList && len(parent.Children[i].Children) > 0 {
				childSymbolNode := parent.Children[i].Children[0]
				if childSymbolNode.Type == reader.NodeSymbol && (childSymbolNode.Value == "catch" || childSymbolNode.Value == "finally") {
					isTryBody = false
					break
				}
			}
		}
		if isTryBody && doNodeIndex >= 1 { // 'do' está no bloco try principal
			isRedundant = true
		}
	case "catch": // Parent é (catch Tipo Var corpo-real...)
		if doNodeIndex >= 3 { // 'do' é parte do corpo-real
			isRedundant = true
		}
	case "finally": // Parent é (finally corpo-real...)
		if doNodeIndex >= 1 { // 'do' é parte do corpo-real
			isRedundant = true
		}
	case "cond", "condp":
		// (cond teste1 expr1 teste2 expr2 ...) : 'do' como exprN
		// Teste está em 1, 3, 5... Expr está em 2, 4, 6...
		// Para cond, condp: se doNodeIndex é par e >= 2
		if parentSymbol == "cond" || parentSymbol == "condp" {
			if doNodeIndex >= 2 && doNodeIndex%2 == 0 {
				isRedundant = true
			}
		}
	case "case":
		// (case expr testval1 expr1 testval2 expr2 ...) : 'do' como exprN
		// expr está em 0, testval1 em 1, expr1 em 2.
		// Então, se doNodeIndex é par e >= 2
		if doNodeIndex >= 2 && doNodeIndex%2 == 0 {
			isRedundant = true
		}
	}

	if isRedundant {
		return r.createFinding(node, parent, redundantInForm, filepath)
	}

	return nil
}

func (r *RedundantDoBlockRule) createFinding(doNode, parentNode *reader.RichNode, parentFormName string, filepath string) *Finding {
	// É possível que o bloco 'do' seja a *única* expressão em uma forma implicit-do.
	// ex: (defn foo [] (do (println "bar")))
	// Neste caso, o 'do' ainda é redundante.
	// Número de expressões reais dentro do bloco 'do'
	numDoChildren := len(doNode.Children) - 1 // -1 para o próprio símbolo 'do'

	message := fmt.Sprintf("Bloco `do` redundante encontrado. A forma `%s` circundante já fornece um `do` implícito para suas expressões de corpo.", parentFormName)
	if numDoChildren == 0 {
		message = fmt.Sprintf("Bloco `do` redundante e vazio encontrado dentro de `%s`. Considere removê-lo completamente.", parentFormName)
	} else if numDoChildren == 1 {
		message = fmt.Sprintf("Bloco `do` redundante com uma única expressão encontrado dentro de `%s`. O wrapper `do` é desnecessário aqui.", parentFormName)
	}

	return &Finding{
		RuleID:   r.Meta().ID,
		Message:  message,
		Filepath: filepath,
		Location: doNode.Location,
		Severity: r.Meta().Severity,
	}
}

func init() {
	RegisterRule(&RedundantDoBlockRule{})
}
