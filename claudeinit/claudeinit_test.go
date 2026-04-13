package claudeinit

import (
	"context"
	"testing"

	"goc/ccb-engine/settingsfile"
)

func TestInit_idempotent(t *testing.T) {
	ResetForTesting()
	settingsfile.ResetEnsureForTesting()

	opts := Options{NonInteractive: true}
	if err := Init(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if err := Init(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
}

func TestRunCleanupsReverseOrder(t *testing.T) {
	var order []int
	RegisterCleanup(func() { order = append(order, 1) })
	RegisterCleanup(func() { order = append(order, 2) })
	RegisterCleanup(func() { order = append(order, 3) })
	RunCleanups()
	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Fatalf("got %v want [3 2 1]", order)
	}
	RunCleanups() // second call: no-op
}

func TestParseGitRemote(t *testing.T) {
	cases := []struct {
		in   string
		want string // host/owner/name
	}{
		{"git@github.com:foo/bar.git", "github.com/foo/bar"},
		{"https://github.com/foo/bar", "github.com/foo/bar"},
		{"https://github.com/foo/bar.git", "github.com/foo/bar"},
		{"git@github.com-work:foo/bar.git", ""},
		{"not a url", ""},
	}
	for _, tc := range cases {
		p := ParseGitRemote(tc.in)
		var got string
		if p != nil {
			got = p.Host + "/" + p.Owner + "/" + p.Name
		}
		if tc.want == "" {
			if p != nil {
				t.Fatalf("%q: want nil, got %#v", tc.in, p)
			}
			continue
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDumpState_schema(t *testing.T) {
	ResetForTesting()
	settingsfile.ResetEnsureForTesting()

	if err := Init(context.Background(), Options{NonInteractive: true}); err != nil {
		t.Fatal(err)
	}
	d := DumpState()
	if d.Schema != dumpSchema {
		t.Fatalf("schema %q", d.Schema)
	}
	if d.Runtime != "go" {
		t.Fatalf("runtime %q", d.Runtime)
	}
}
