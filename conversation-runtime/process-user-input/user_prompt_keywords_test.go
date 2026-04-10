package processuserinput

import "testing"

func TestMatchesNegativeKeyword(t *testing.T) {
	if !MatchesNegativeKeyword("This is awful") {
		t.Fatal("expected awful")
	}
	if MatchesNegativeKeyword("hello") {
		t.Fatal("expected plain hello false")
	}
}

func TestMatchesKeepGoingKeyword(t *testing.T) {
	if !MatchesKeepGoingKeyword("continue") {
		t.Fatal("continue alone")
	}
	if !MatchesKeepGoingKeyword("please keep going") {
		t.Fatal("keep going phrase")
	}
	if MatchesKeepGoingKeyword("continued story") {
		t.Fatal("continue as substring only when whole prompt is continue in TS; 'continued' should be false")
	}
}
