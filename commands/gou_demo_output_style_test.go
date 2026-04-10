package commands

import (
	"strings"
	"testing"
)

func TestResolveGouDemoOutputStyle_envPairWins(t *testing.T) {
	n, p := ResolveGouDemoOutputStyle("Custom", "prompt-body", "Learning")
	if n != "Custom" || p != "prompt-body" {
		t.Fatalf("got %q / %q", n, p)
	}
}

func TestResolveGouDemoOutputStyle_defaultEmpty(t *testing.T) {
	n, p := ResolveGouDemoOutputStyle("", "", "default")
	if n != "" || p != "" {
		t.Fatalf("got %q / %q", n, p)
	}
}

func TestResolveGouDemoOutputStyle_learningBuiltin(t *testing.T) {
	n, p := ResolveGouDemoOutputStyle("", "", "Learning")
	if n != "Learning" {
		t.Fatalf("name %q", n)
	}
	if !strings.Contains(p, "Learn by Doing") {
		t.Fatalf("prompt missing marker, len=%d", len(p))
	}
}

func TestResolveGouDemoOutputStyle_explanatoryBuiltin(t *testing.T) {
	n, p := ResolveGouDemoOutputStyle("", "", "Explanatory")
	if n != "Explanatory" {
		t.Fatalf("name %q", n)
	}
	if !strings.Contains(p, "Explanatory Style Active") {
		t.Fatalf("prompt missing marker")
	}
}

func TestResolveGouDemoOutputStyle_unknownKey(t *testing.T) {
	n, p := ResolveGouDemoOutputStyle("", "", "NoSuchStyle")
	if n != "" || p != "" {
		t.Fatalf("got %q / %q", n, p)
	}
}
