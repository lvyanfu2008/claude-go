package compactquerysource

import "testing"

func TestMainThreadLike(t *testing.T) {
	if !MainThreadLike("") {
		t.Fatal("empty => main")
	}
	if !MainThreadLike(`"repl_main_thread:outputStyle:custom"`) {
		t.Fatal("json string repl_main_thread* => main")
	}
	if !MainThreadLike("sdk") {
		t.Fatal("sdk => main")
	}
	if !MainThreadLike(`"sdk"`) {
		t.Fatal("json-encoded sdk => main")
	}
	if MainThreadLike(`"agent:builtin"`) {
		t.Fatal("agent:* => not main-thread-like for compact cleanup")
	}
}
