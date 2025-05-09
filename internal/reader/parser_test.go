package reader

import (
	"path/filepath"
	"testing"

	"github.com/cespare/goclj/parse"
)

func TestParseFile_Valid(t *testing.T) {

	examplePath := filepath.Join(".", "example.clj")

	tree, err := ParseFile(examplePath)

	if err != nil {
		t.Fatalf("ParseFile(%q) returned an unexpected error: %v", examplePath, err)
	}

	if tree == nil {
		t.Fatalf("ParseFile(%q) returned a nil tree without an error", examplePath)
	}

	if len(tree.Roots) == 0 {
		t.Fatalf("ParseFile(%q) returned a tree with no root nodes", examplePath)
	}

	defns := FindTopLevelDefns(tree)
	expectedDefnCount := 2
	if len(defns) != expectedDefnCount {
		t.Errorf("FindTopLevelDefns: expected %d 'defn' nodes, got %d", expectedDefnCount, len(defns))
	}

	if len(defns) == expectedDefnCount {

		foundNames := make(map[string]bool)
		for _, defnNode := range defns {
			if len(defnNode.Nodes) > 1 {
				if nameNode, ok := defnNode.Nodes[1].(*parse.SymbolNode); ok {
					foundNames[nameNode.Val] = true
				}
			}
		}
		if !foundNames["greet"] {
			t.Errorf("FindTopLevelDefns: did not find function 'greet'")
		}
		if !foundNames["-main"] {
			t.Errorf("FindTopLevelDefns: did not find function '-main'")
		}
	}

}
