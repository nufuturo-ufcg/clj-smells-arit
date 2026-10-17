package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

func analyzeSource(t *testing.T, source string) AnalysisResult {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.clj")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeFile(path, &config.Config{
		EnabledRules:  map[string]bool{},
		EnabledGroups: map[string]bool{"clojure-specific": false},
		RuleConfig:    map[string]config.RuleSettings{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func callHeadResolution(result AnalysisResult, name string) *reader.SymbolResolution {
	var found *reader.SymbolResolution
	var walk func(*reader.RichNode)
	walk = func(node *reader.RichNode) {
		if node == nil || found != nil {
			return
		}
		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol &&
			node.Children[0].Value == name && node.Children[0].Resolution != nil {
			found = node.Children[0].Resolution
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range result.RichRoots[1:] {
		walk(root)
	}
	return found
}

func TestReferClojureRemovesCoreResolution(t *testing.T) {
	cases := []struct {
		name, source, symbol string
		wantCore             bool
	}{
		{"plain ns keeps core", "(ns a)\n(defn f [x] (> x 0))", ">", true},
		{"exclude removes core", "(ns a (:refer-clojure :exclude [>]))\n(defn f [x] (> x 0))", ">", false},
		{"only removes unlisted core", "(ns a (:refer-clojure :only [inc]))\n(defn f [x] (> x 0))", ">", false},
		{"only keeps listed core", "(ns a (:refer-clojure :only [>]))\n(defn f [x] (> x 0))", ">", true},
		{"rename removes original", "(ns a (:refer-clojure :rename {> gt}))\n(defn f [x] (> x 0))", ">", false},
		{"special form cannot be excluded", "(ns a (:refer-clojure :exclude [if]))\n(defn f [x] (if x 1 2))", "if", true},
		{"known colliding ns replaces its names", "(ns a (:require [clojure.string :refer :all]))\n(defn f [x] (reverse x))", "reverse", false},
		{"known ns leaves other core names", "(ns a (:require [clojure.string :refer :all]))\n(defn f [x] (> x 0))", ">", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolution := callHeadResolution(analyzeSource(t, tc.source), tc.symbol)
			isCore := resolution != nil && resolution.Kind == reader.ResolutionClojureCore
			if isCore != tc.wantCore {
				t.Fatalf("%s resolved as %+v, want core=%t", tc.symbol, resolution, tc.wantCore)
			}
		})
	}
}

func TestWholeNamespaceReferralAmbiguity(t *testing.T) {
	cases := []struct {
		name, source string
		want         bool
	}{
		{"no referral", "(ns a (:require [clojure.string :as str]))", false},
		{"unknown :use", "(ns a (:use [my.vec-math]))", true},
		{"unknown :refer :all", "(ns a (:require [my.vec-math :refer :all]))", true},
		{"known no-collision :use", "(ns a (:use clojure.test))", false},
		{"known :refer :all", "(ns a (:require [clojure.test :refer :all]))", false},
		{":use limited by :only", "(ns a (:use [my.lib :only [helper]]))", false},
		{"prefix list", "(ns a (:use [my random util]))", true},
		{"top-level use", "(ns a)\n(use 'my.lib)", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := analyzeSource(t, tc.source).GlobalScope.CoreNamesAmbiguous(); got != tc.want {
				t.Fatalf("ambiguous = %t, want %t", got, tc.want)
			}
		})
	}
}

// Multi-arity fn/defn: every arity body is resolved and every arity's params
// are local, so a param named like a core function is not clojure.core.
func TestMultiArityBodiesAndParamsResolve(t *testing.T) {
	result := analyzeSource(t, "(ns a)\n(def f (fn g ([count] (count 1)) ([count x] (clojure.lang.RT/assoc x 1 2))))\n(defn h ([m] (clojure.lang.RT/iter m)) ([m n] n))\n")
	var heads []*reader.SymbolResolution
	var walk func(*reader.RichNode)
	walk = func(node *reader.RichNode) {
		if node == nil {
			return
		}
		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			switch node.Children[0].Value {
			case "count", "clojure.lang.RT/assoc", "clojure.lang.RT/iter":
				heads = append(heads, node.Children[0].Resolution)
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range result.RichRoots[1:] {
		walk(root)
	}
	if len(heads) != 3 {
		t.Fatalf("expected 3 call heads, got %d", len(heads))
	}
	for _, resolution := range heads {
		if resolution == nil {
			t.Fatalf("a call head inside a multi-arity body was not resolved")
		}
	}
	if heads[0].Kind != reader.ResolutionLocal {
		t.Errorf("param `count` must shadow clojure.core/count, got %+v", heads[0])
	}
	if heads[1].Kind != reader.ResolutionJavaStatic || heads[2].Kind != reader.ResolutionJavaStatic {
		t.Errorf("RT calls must resolve as Java statics: %+v %+v", heads[1], heads[2])
	}
}
