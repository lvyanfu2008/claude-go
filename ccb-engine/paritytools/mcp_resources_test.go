package paritytools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListMcpResources_emptyDataArray(t *testing.T) {
	t.Parallel()
	out, isErr, err := ListMcpResourcesFromJSON([]byte(`{}`))
	if err != nil || isErr {
		t.Fatalf("list: err=%v isErr=%v", err, isErr)
	}
	var wrap map[string]any
	if err := json.Unmarshal([]byte(out), &wrap); err != nil {
		t.Fatal(err)
	}
	data, ok := wrap["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %#v", wrap["data"])
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %#v", data)
	}
}

func TestListMcpResources_unknownServer(t *testing.T) {
	t.Parallel()
	_, isErr, err := ListMcpResourcesFromJSON([]byte(`{"server":"nope"}`))
	if err == nil || !isErr {
		t.Fatalf("expected error, err=%v isErr=%v", err, isErr)
	}
	if !strings.Contains(err.Error(), `Server "nope" not found`) {
		t.Fatalf("got %v", err)
	}
}

func TestReadMcpResource_unknownServer(t *testing.T) {
	t.Parallel()
	_, isErr, err := ReadMcpResourceFromJSON([]byte(`{"server":"x","uri":"file:///a"}`))
	if err == nil || !isErr {
		t.Fatalf("expected error, err=%v isErr=%v", err, isErr)
	}
	if !strings.Contains(err.Error(), `Server "x" not found`) {
		t.Fatalf("got %v", err)
	}
}

func TestReadMcpResource_requiresFields(t *testing.T) {
	t.Parallel()
	_, _, err := ReadMcpResourceFromJSON([]byte(`{"server":"","uri":"u"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	_, _, err = ReadMcpResourceFromJSON([]byte(`{"server":"s","uri":""}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
