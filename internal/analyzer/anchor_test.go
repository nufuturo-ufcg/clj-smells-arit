package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thlaurentino/arit/internal/config"
	_ "github.com/thlaurentino/arit/internal/rules/clojurespecific"
)

// anchorsFor returns rule|anchor -> count for the verbose-checks findings of a source.
func anchorsFor(t *testing.T, source string) (map[string]int, []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.clj")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeFile(path, &config.Config{EnabledRules: map[string]bool{}, EnabledGroups: map[string]bool{}, RuleConfig: map[string]config.RuleSettings{}})
	if err != nil {
		t.Fatal(err)
	}
	anchors := map[string]int{}
	namespaces := []string{}
	for _, finding := range result.Findings {
		if finding.RuleID == "verbose-checks" {
			anchors[finding.Anchor]++
			namespaces = append(namespaces, finding.Namespace)
		}
	}
	return anchors, namespaces
}

// The anchor is the identity of the enclosing top-level form: editing the
// function body, the ns form, or moving the form must not change it, while the
// AST fingerprint of the reported node may change.
func TestAnchorStableAcrossEdits(t *testing.T) {
	before := "(ns app.core)\n\n(defn foo [x]\n  (if (= 0 x) :zero :other))\n\n(defn bar [y]\n  (= 0 y))\n"
	after := "(ns app.core\n  (:require [clojure.string :as str]))\n\n(def extra 1)\n\n(defn foo [x]\n  (println \"edited\" x)\n  (if (= 0 (inc x)) :zero :other))\n\n(defn bar [y]\n  (= 0 y))\n"
	a, ns := anchorsFor(t, before)
	b, _ := anchorsFor(t, after)
	for _, anchor := range []string{"defn:foo", "defn:bar"} {
		if a[anchor] != 1 || b[anchor] != 1 {
			t.Errorf("anchor %q: before=%d after=%d (all before=%v after=%v)", anchor, a[anchor], b[anchor], a, b)
		}
	}
	for _, n := range ns {
		if n != "app.core" {
			t.Errorf("namespace = %q, want app.core", n)
		}
	}
}

func TestAnchorReaderConditionalAndDefmethod(t *testing.T) {
	source := "(ns app.shapes)\n\n#?(:clj (defn f [x] (= 0 x)))\n\n(defmethod area :circle [c] (= 0 (:r c)))\n(defmethod area :square [s] (= 0 (:side s)))\n"
	path := filepath.Join(t.TempDir(), "a.cljc")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeFile(path, &config.Config{EnabledRules: map[string]bool{}, EnabledGroups: map[string]bool{}, RuleConfig: map[string]config.RuleSettings{}})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, finding := range result.Findings {
		if finding.RuleID == "verbose-checks" {
			got[finding.Anchor] = true
		}
	}
	for _, want := range []string{"defn:f", "defmethod:area:Keyword(:circle)", "defmethod:area:Keyword(:square)"} {
		if !got[want] {
			t.Errorf("missing anchor %q in %v", want, got)
		}
	}
}
