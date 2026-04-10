package appstate

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNotificationSnapshot_roundTrip(t *testing.T) {
	n := 5000
	s := NotificationSnapshot{
		Key:         "k",
		Invalidates: []string{"a"},
		Priority:    "high",
		TimeoutMs:   &n,
		Text:        "hello",
		Color:       "accent",
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back NotificationSnapshot
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Key != s.Key || back.Text != s.Text || back.Priority != s.Priority {
		t.Fatalf("%+v", back)
	}
	if back.TimeoutMs == nil || *back.TimeoutMs != n {
		t.Fatalf("timeout: %+v", back.TimeoutMs)
	}
}

func TestElicitationRequestEventSnapshot_roundTrip(t *testing.T) {
	done := true
	s := ElicitationRequestEventSnapshot{
		ServerName: "srv",
		RequestID:  json.RawMessage(`42`),
		Params:     json.RawMessage(`{"mode":"form"}`),
		WaitingState: &ElicitationWaitingState{
			ActionLabel: "Retry",
		},
		Completed: &done,
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back ElicitationRequestEventSnapshot
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ServerName != "srv" || string(back.RequestID) != "42" {
		t.Fatalf("%+v", back)
	}
}

func TestLoadedPluginData_roundTrip(t *testing.T) {
	p := LoadedPluginData{
		Name:       "p",
		Manifest:   json.RawMessage(`{"name":"p"}`),
		Path:       "/x",
		Source:     "s",
		Repository: "s",
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var back LoadedPluginData
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Name != "p" || string(back.Manifest) != `{"name":"p"}` {
		t.Fatalf("%+v", back)
	}
}

func TestComputerUseMcpState_roundTrip(t *testing.T) {
	st := ComputerUseMcpState{
		AllowedApps: []ComputerUseMcpAllowedApp{
			{BundleID: "b", DisplayName: "B", GrantedAt: 1},
		},
		HiddenDuringTurn: []string{"x"},
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	var back ComputerUseMcpState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.AllowedApps) != 1 || back.AllowedApps[0].BundleID != "b" {
		t.Fatalf("%+v", back)
	}
}

func TestReplContextState_roundTrip(t *testing.T) {
	st := ReplContextState{
		RegisteredTools: map[string]ReplRegisteredToolSnapshot{
			"t": {
				Name:        "t",
				Description: "d",
				Schema:      json.RawMessage(`{"type":"object"}`),
			},
		},
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	var back ReplContextState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.RegisteredTools) != 1 || back.RegisteredTools["t"].Name != "t" {
		t.Fatalf("%+v", back)
	}
}

func TestTeamContextState_roundTrip(t *testing.T) {
	leader := true
	st := TeamContextState{
		TeamName:     "team",
		TeamFilePath: "/p",
		LeadAgentID:  "a1",
		IsLeader:     &leader,
		Teammates: map[string]TeamTeammateInfo{
			"w1": {
				Name:            "w",
				TmuxSessionName: "s",
				TmuxPaneID:      "%1",
				Cwd:             "/c",
				SpawnedAt:       99,
			},
		},
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	var back TeamContextState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.TeamName != "team" || back.Teammates["w1"].TmuxPaneID != "%1" {
		t.Fatalf("%+v", back)
	}
}

func TestNormalizeAppState_replAndTeamContext(t *testing.T) {
	a := AppState{
		ReplContext: &ReplContextState{},
		TeamContext: &TeamContextState{},
	}
	NormalizeAppState(&a)
	if a.ReplContext.RegisteredTools == nil {
		t.Fatal("repl registeredTools")
	}
	if a.TeamContext.Teammates == nil {
		t.Fatal("team teammates")
	}
}

func TestNormalizeAppState_tasksRegistryOverlays(t *testing.T) {
	a := AppState{}
	NormalizeAppState(&a)
	if a.Tasks == nil || a.AgentNameRegistry == nil || a.ActiveOverlays == nil {
		t.Fatalf("tasks=%v registry=%v overlays=%v", a.Tasks, a.AgentNameRegistry, a.ActiveOverlays)
	}
}

func TestNormalizeAppState_pluginsNotificationsElicitationComputerUse(t *testing.T) {
	a := AppState{}
	NormalizeAppState(&a)
	if a.Plugins.Enabled == nil || a.Plugins.Disabled == nil || a.Plugins.Errors == nil {
		t.Fatal("plugins slices")
	}
	if a.Notifications.Queue == nil {
		t.Fatal("notifications queue")
	}
	if a.Elicitation.Queue == nil {
		t.Fatal("elicitation queue")
	}
	a.ComputerUseMcpState = &ComputerUseMcpState{}
	NormalizeAppState(&a)
	if a.ComputerUseMcpState.HiddenDuringTurn == nil || a.ComputerUseMcpState.AllowedApps == nil {
		t.Fatal("computer use slices")
	}
	if !reflect.DeepEqual(a.ComputerUseMcpState.HiddenDuringTurn, []string{}) {
		t.Fatalf("hidden: %#v", a.ComputerUseMcpState.HiddenDuringTurn)
	}
}
