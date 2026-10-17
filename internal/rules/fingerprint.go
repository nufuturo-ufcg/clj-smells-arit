package rules

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// normalizeNode recursively serializes an AST node into a canonical, deterministic string
// that is invariant to:
//   - whitespace and formatting
//   - absolute line/column positions
//   - comments
//   - doc-strings (first string literal child of defn/defmacro)
//
// This canonical form is then hashed (SHA-256) to produce a stable identity
// for each smell instance across different commits of the same repository.
func normalizeNode(node *reader.RichNode, depth int) string {
	if node == nil {
		return ""
	}

	// Skip comments and newlines entirely — they are not semantically relevant
	if node.Type == reader.NodeComment || node.Type == reader.NodeNewline {
		return ""
	}

	// Leaf nodes: emit type + value (normalized)
	if len(node.Children) == 0 {
		val := strings.TrimSpace(node.Value)
		return fmt.Sprintf("%s(%s)", node.Type, val)
	}

	// For composite nodes, recursively normalize children.
	// We skip doc-strings: the first string child of a top-level defn/defmacro.
	var parts []string
	for i, child := range node.Children {
		if child == nil {
			continue
		}
		// Skip doc-strings: second child of (defn ...) or (defmacro ...) if it's a string
		if depth == 0 && i == 2 && child.Type == reader.NodeString {
			headIsDefn := len(node.Children) > 0 &&
				node.Children[0] != nil &&
				(node.Children[0].Value == "defn" ||
					node.Children[0].Value == "defmacro" ||
					node.Children[0].Value == "defn-" ||
					node.Children[0].Value == "defprotocol" ||
					node.Children[0].Value == "deftype" ||
					node.Children[0].Value == "defrecord")
			if headIsDefn {
				continue
			}
		}

		normalized := normalizeNode(child, depth+1)
		if normalized != "" {
			parts = append(parts, normalized)
		}
	}

	inner := strings.Join(parts, " ")

	switch node.Type {
	case reader.NodeList:
		return fmt.Sprintf("L[%s]", inner)
	case reader.NodeVector:
		return fmt.Sprintf("V[%s]", inner)
	case reader.NodeMap:
		return fmt.Sprintf("M[%s]", inner)
	case reader.NodeSet:
		return fmt.Sprintf("S[%s]", inner)
	case reader.NodeQuote:
		return fmt.Sprintf("Q[%s]", inner)
	case reader.NodeSyntaxQuote:
		return fmt.Sprintf("SQ[%s]", inner)
	case reader.NodeUnquote:
		return fmt.Sprintf("UQ[%s]", inner)
	case reader.NodeUnquoteSplice:
		return fmt.Sprintf("UQS[%s]", inner)
	case reader.NodeFnLiteral:
		return fmt.Sprintf("FN[%s]", inner)
	case reader.NodeDeref:
		return fmt.Sprintf("DEREF[%s]", inner)
	case reader.NodeMetadata:
		// Metadata is structurally relevant but should not affect identity of
		// the form itself for lifecycle tracking. We include it for precision.
		return fmt.Sprintf("META[%s]", inner)
	default:
		return fmt.Sprintf("%s[%s]", node.Type, inner)
	}
}

// ComputeFingerprint generates a stable, position-independent SHA-256 fingerprint
// for the given AST node. The fingerprint uniquely identifies the logical structure
// of the smell instance and can be used to track it across commits.
//
// Two smell instances with the same fingerprint represent the same logical entity,
// even if they appear on different line numbers due to code additions/removals above them.
func ComputeFingerprint(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	canonical := normalizeNode(node, 0)
	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash[:8]) // First 8 bytes → 16 hex chars (sufficient for uniqueness)
}

// ComputeAnchor identifies the top-level form that contains a finding:
// "ns:my.app", "defn:foo", "defmethod:area:<dispatch>", or "<head>:" for an
// anonymous form. Unlike the AST fingerprint, the anchor does not
// change when code inside the form is edited, so lifecycle tracking can
// follow a smell across commits that touch the surrounding function.
func ComputeAnchor(node *reader.RichNode, ancestors []*reader.RichNode) string {
	top := node
	if len(ancestors) > 0 {
		top = ancestors[0]
		// #?(:clj (defn f ...)): the platform branch is the real top form.
		if top != nil && (top.Type == reader.NodeReaderCond || top.Type == reader.NodeReaderCondSplice) {
			if len(ancestors) > 1 {
				top = ancestors[1]
			} else {
				top = node
			}
		}
	}
	if top == nil {
		return ""
	}
	if top.Type != reader.NodeList || len(top.Children) == 0 || top.Children[0] == nil ||
		top.Children[0].Type != reader.NodeSymbol {
		return fmt.Sprintf("%s:", top.Type)
	}
	head := strings.TrimPrefix(top.Children[0].Value, "clojure.core/")
	if len(top.Children) > 1 && top.Children[1] != nil &&
		(top.Children[1].Type == reader.NodeSymbol || top.Children[1].Type == reader.NodeKeyword) {
		name := top.Children[1].Value
		if head == "defmethod" && len(top.Children) > 2 && top.Children[2] != nil {
			// Methods of one multimethod differ by dispatch value.
			return fmt.Sprintf("%s:%s:%s", head, name, normalizeNode(top.Children[2], 1))
		}
		return fmt.Sprintf("%s:%s", head, name)
	}
	// Anonymous top-level form, e.g. (use ...) or (try ...): only its head.
	// Occurrences under the same head are told apart by fingerprint and order.
	return fmt.Sprintf("%s:", head)
}
