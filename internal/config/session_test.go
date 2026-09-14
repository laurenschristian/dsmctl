package config

import (
	"path/filepath"
	"testing"
)

func TestSessionRoundTrip(t *testing.T) {
	t.Setenv("DSM_CONFIG", filepath.Join(t.TempDir(), "config.yaml"))
	if s := LoadSession(); s.SID != "" {
		t.Fatal("expected empty session")
	}
	if err := SaveSession(Session{SID: "abc", SynoToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	got := LoadSession()
	if got.SID != "abc" || got.SynoToken != "tok" {
		t.Fatalf("bad session %+v", got)
	}
	if !filepath.IsAbs(SessionPath()) {
		t.Fatal("session path not absolute")
	}
	if err := ClearSession(); err != nil {
		t.Fatal(err)
	}
	if LoadSession().SID != "" {
		t.Fatal("session not cleared")
	}
	// Clearing a missing session is a no-op.
	if err := ClearSession(); err != nil {
		t.Fatal(err)
	}
}
