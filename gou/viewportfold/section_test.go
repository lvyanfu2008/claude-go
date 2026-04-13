package viewportfold

import (
	"strings"
	"testing"
)

func TestToggleAll(t *testing.T) {
	s := []Section{{Title: "A", Collapsed: false}, {Title: "B", Collapsed: true}}
	ToggleAll(s)
	if !s[0].Collapsed || s[1].Collapsed {
		t.Fatalf("after first toggle want A collapsed B expanded: %+v", s)
	}
	ToggleAll(s)
	if s[0].Collapsed || !s[1].Collapsed {
		t.Fatalf("after second toggle want A expanded B collapsed: %+v", s)
	}
}

func TestAppendSections_expandedAndCollapsed(t *testing.T) {
	var b strings.Builder
	AppendSections(&b, []Section{
		{Title: "T1", Content: "body1", Collapsed: false},
		{Title: "T2", Content: "body2", Collapsed: true},
	})
	out := b.String()
	if !strings.Contains(out, "▼ T1") || !strings.Contains(out, "body1") {
		t.Fatalf("expanded section missing: %q", out)
	}
	if !strings.Contains(out, "▶ T2") || !strings.Contains(out, "[内容已折叠...]") {
		t.Fatalf("collapsed section missing: %q", out)
	}
}
