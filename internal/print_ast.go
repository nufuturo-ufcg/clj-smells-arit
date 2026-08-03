package main

import (
	"encoding/json"
	"fmt"
	"github.com/thlaurentino/arit/internal/reader"
)

func main() {
	tree, err := reader.ParseFile("docs/expanded_smells_catalog/23_private_multimethods/complex_02.clj")
	if err != nil {
		panic(err)
	}
	ast, _ := reader.BuildRichTree(tree)
	j, _ := json.MarshalIndent(ast, "", "  ")
	fmt.Println(string(j))
}
