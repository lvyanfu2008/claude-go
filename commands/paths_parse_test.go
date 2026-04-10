package commands

import (
	"reflect"
	"testing"
)

func TestSplitPathInFrontmatter_BracesAndComma(t *testing.T) {
	got := SplitPathInFrontmatter("{a,b}/{c,d}")
	want := []string{"a/c", "a/d", "b/c", "b/d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestParseSkillPaths_AllStarReturnsNil(t *testing.T) {
	if ParseSkillPaths("**") != nil {
		t.Fatal("expected nil for ** only")
	}
	if ParseSkillPaths([]interface{}{"**", "**"}) != nil {
		t.Fatal("expected nil for all ** list")
	}
}

func TestParseSkillPaths_TrimsGlobSuffix(t *testing.T) {
	got := ParseSkillPaths("src/**")
	if len(got) != 1 || got[0] != "src" {
		t.Fatalf("got %#v", got)
	}
}
